import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../engine/client.dart';
import '../files/local_file_actions.dart';
import '../layout/chat_layout.dart';
import '../theme/dudka_theme.dart';
import '../update/update_manager.dart';
import '../widgets/step_progress.dart';
import 'settings_nick_screen.dart';

/// Chat shell: DESIGN.md charcoal + adaptive dual-pane / peer strip (P063–P070).
class ChatScreen extends StatefulWidget {
  const ChatScreen({
    super.key,
    required this.client,
    this.pollInterval = const Duration(seconds: 1),
    this.pickFiles,
    this.revealFile,
    this.updater,
  });

  final EngineClient client;
  final Duration pollInterval;
  final LocalFilesPicker? pickFiles;
  final DownloadedFileRevealer? revealFile;
  final UpdateController? updater;

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _compose = TextEditingController();
  final List<LocalFileBytes> _pendingFiles = [];
  ChatSnapshot? _snap;
  Object? _error;
  String? _sendError;
  String? _seekError;
  String? _fileError;
  bool _loading = true;
  bool _sending = false;
  bool _seeking = false;
  bool _picking = false;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _refresh();
    widget.updater?.start();
    if (widget.pollInterval > Duration.zero) {
      _timer = Timer.periodic(widget.pollInterval, (_) => _refresh());
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    _compose.dispose();
    super.dispose();
  }

  Future<void> _refresh() async {
    try {
      final snap = await widget.client.fetchSnapshot();
      if (!mounted) return;
      setState(() {
        _snap = snap;
        _error = null;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e;
        _loading = false;
      });
    }
  }

  Future<void> _blow() async {
    if (_sending) return;
    final text = _compose.text.trim();
    final files = List<LocalFileBytes>.of(_pendingFiles);
    if (text.isEmpty && files.isEmpty) return;
    setState(() {
      _sending = true;
      _sendError = null;
      _fileError = null;
    });
    try {
      if (text.isNotEmpty) {
        await widget.client.sendText(text);
        _compose.clear();
      }
      for (final file in files) {
        await widget.client.announceFile(
          name: file.name,
          mime: file.mime,
          content: file.bytes,
        );
        if (mounted) {
          setState(() => _pendingFiles.remove(file));
        }
      }
      await _refresh();
    } catch (e) {
      if (!mounted) return;
      setState(() => _sendError = e.toString());
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  Future<void> _seek() async {
    if (_seeking) return;
    setState(() {
      _seeking = true;
      _seekError = null;
    });
    try {
      await widget.client.startScan();
      await _refresh();
    } catch (e) {
      if (!mounted) return;
      setState(() => _seekError = e.toString());
    } finally {
      if (mounted) setState(() => _seeking = false);
    }
  }

  Future<void> _openSettings() async {
    final current = _snap?.me.name ?? '';
    final updated = await Navigator.of(context).push<String>(
      MaterialPageRoute(
        builder: (_) =>
            SettingsNickScreen(client: widget.client, initialNick: current),
      ),
    );
    if (!mounted) return;
    if (updated != null) {
      await _refresh();
    }
  }

  Future<void> _pickFiles() async {
    if (_picking) return;
    setState(() {
      _picking = true;
      _fileError = null;
    });
    try {
      final picker = widget.pickFiles ?? pickLocalFiles;
      final picked = await picker();
      if (picked.isEmpty || !mounted) return;
      setState(() => _pendingFiles.addAll(picked));
    } catch (e) {
      if (!mounted) return;
      setState(() => _fileError = e.toString());
    } finally {
      if (mounted) setState(() => _picking = false);
    }
  }

  Future<void> _revealFile(String path) async {
    setState(() => _fileError = null);
    try {
      await (widget.revealFile ?? revealDownloadedFile)(path);
    } catch (e) {
      if (!mounted) return;
      setState(() => _fileError = e.toString());
    }
  }

  Future<void> _fetchFile(String fileId) async {
    setState(() => _fileError = null);
    try {
      await widget.client.startFetch(fileId);
      await _refresh();
    } catch (e) {
      if (!mounted) return;
      setState(() => _fileError = e.toString());
    }
  }

  Future<void> _cancelFile(String fileId) async {
    setState(() => _fileError = null);
    try {
      await widget.client.cancelFetch(fileId);
      await _refresh();
    } catch (e) {
      if (!mounted) return;
      setState(() => _fileError = e.toString());
    }
  }

  Future<void> _activateUpdate() async {
    final updater = widget.updater;
    if (updater == null) return;
    try {
      await updater.activate();
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Не удалось обновить: $error')));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 16),
          child: SelectionArea(child: _body()),
        ),
      ),
    );
  }

  Widget _body() {
    if (_loading && _snap == null) {
      return const Center(
        child: CircularProgressIndicator(key: Key('chat-loading')),
      );
    }
    if (_error != null && _snap == null) {
      return Text('движок недоступен\n$_error', key: const Key('chat-error'));
    }
    final snap = _snap!;
    final state = chatNetworkState(
      network: snap.network,
      peerCount: snap.remotePeerCount,
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          key: const Key('chat-header'),
          children: [
            Expanded(
              child: Text(
                formatStatusStrip(snap),
                key: const Key('chat-status'),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: DudkaType.mono(
                  size: 13,
                  weight: FontWeight.w700,
                  letterSpacing: 1.2,
                ),
              ),
            ),
            if (widget.updater != null)
              AnimatedBuilder(
                animation: widget.updater!,
                builder: (context, _) {
                  final update = widget.updater!.snapshot;
                  if (!update.isReady) return const SizedBox.shrink();
                  return Padding(
                    padding: const EdgeInsets.only(left: 8),
                    child: FilledButton.icon(
                      key: const Key('update-ready'),
                      onPressed: _activateUpdate,
                      icon: const Icon(Icons.system_update_alt, size: 18),
                      label: Text('АПДЕЙТ ${update.version}'),
                    ),
                  );
                },
              ),
            IconButton(
              key: const Key('chat-settings'),
              tooltip: 'Настройки',
              icon: const Icon(Icons.settings_outlined),
              onPressed: _openSettings,
            ),
          ],
        ),
        const SizedBox(height: 6),
        Expanded(
          child: LayoutBuilder(
            builder: (context, constraints) {
              final wide = isWideChatLayout(constraints.maxWidth);
              if (wide) {
                return _wideBody(state, snap);
              }
              return _narrowBody(state, snap);
            },
          ),
        ),
      ],
    );
  }

  Widget _wideBody(String state, ChatSnapshot snap) {
    return Row(
      key: const Key('chat-layout-wide'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        SizedBox(
          width: 220,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text('СОСЕДИ', style: DudkaType.label()),
              const SizedBox(height: 4),
              Expanded(
                child: Container(
                  key: const Key('chat-peers'),
                  alignment: Alignment.topLeft,
                  decoration: const BoxDecoration(
                    border: Border(
                      top: BorderSide(
                        color: DudkaColors.silkscreenDim,
                        width: 1,
                      ),
                      right: BorderSide(
                        color: DudkaColors.silkscreenDim,
                        width: 1,
                      ),
                    ),
                  ),
                  padding: const EdgeInsets.fromLTRB(0, 8, 12, 0),
                  child: KeyedSubtree(
                    key: const Key('chat-peers-pane'),
                    child: _peersList(state, snap, axis: Axis.vertical),
                  ),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: 12),
        Expanded(child: _feedColumn(snap)),
      ],
    );
  }

  Widget _narrowBody(String state, ChatSnapshot snap) {
    return Column(
      key: const Key('chat-layout-narrow'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text('СОСЕДИ', style: DudkaType.label()),
        const SizedBox(height: 4),
        SizedBox(
          height: state == 'alone' || state == 'no_network' ? 88 : 40,
          child: Container(
            key: const Key('chat-peers'),
            alignment: Alignment.centerLeft,
            decoration: const BoxDecoration(
              border: Border(
                top: BorderSide(color: DudkaColors.silkscreenDim, width: 1),
              ),
            ),
            padding: const EdgeInsets.only(top: 8),
            child: KeyedSubtree(
              key: const Key('chat-peers-strip'),
              child: _peersList(state, snap, axis: Axis.horizontal),
            ),
          ),
        ),
        const SizedBox(height: 8),
        Expanded(child: _feedColumn(snap)),
      ],
    );
  }

  Widget _feedColumn(ChatSnapshot snap) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text('ЛЕНТА', style: DudkaType.label()),
        const SizedBox(height: 4),
        Expanded(
          child: Container(
            key: const Key('chat-feed'),
            alignment: Alignment.topLeft,
            decoration: const BoxDecoration(
              color: DudkaColors.panelDeep,
              border: Border(
                top: BorderSide(color: DudkaColors.silkscreenDim, width: 1),
              ),
            ),
            padding: const EdgeInsets.only(top: 8),
            child: _feedPane(snap),
          ),
        ),
        if (_sendError != null)
          Text(
            _sendError!,
            style: DudkaType.mono(size: 12, color: DudkaColors.danger),
          ),
        if (_fileError != null)
          Text(
            _fileError!,
            style: DudkaType.mono(size: 12, color: DudkaColors.danger),
          ),
        const SizedBox(height: 8),
        _composeRow(),
      ],
    );
  }

  Widget _composeRow() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (_pendingFiles.isNotEmpty) ...[
          SizedBox(
            key: const Key('chat-pending-files'),
            height: 84,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: _pendingFiles.length,
              separatorBuilder: (_, __) => const SizedBox(width: 8),
              itemBuilder: (context, index) =>
                  _pendingFile(_pendingFiles[index], index),
            ),
          ),
          const SizedBox(height: 8),
        ],
        Row(
          children: [
            Expanded(
              child: CallbackShortcuts(
                bindings: {
                  const SingleActivator(LogicalKeyboardKey.enter, meta: true):
                      _blow,
                  const SingleActivator(
                    LogicalKeyboardKey.enter,
                    control: true,
                  ): _blow,
                },
                child: TextField(
                  key: const Key('chat-compose'),
                  controller: _compose,
                  enabled: !_sending,
                  minLines: 1,
                  maxLines: 5,
                  decoration: const InputDecoration(
                    hintText: 'текст или комментарий…',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  textInputAction: TextInputAction.newline,
                ),
              ),
            ),
            const SizedBox(width: 4),
            IconButton(
              key: const Key('chat-file'),
              tooltip: 'Прикрепить файлы',
              onPressed: _picking ? null : _pickFiles,
              icon: _picking
                  ? const SizedBox.square(
                      dimension: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.attach_file),
            ),
            const SizedBox(width: 2),
            IconButton.filled(
              key: const Key('chat-blow'),
              tooltip: 'Отправить · ⌘/Ctrl+Enter',
              onPressed: _sending ? null : _blow,
              icon: _sending
                  ? const SizedBox.square(
                      dimension: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.send),
            ),
          ],
        ),
      ],
    );
  }

  Widget _pendingFile(LocalFileBytes file, int index) {
    return Container(
      key: Key('pending-file-$index'),
      width: 148,
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: DudkaColors.panelDeep,
        border: Border.all(color: DudkaColors.silkscreenDim),
      ),
      child: Row(
        children: [
          SizedBox.square(
            dimension: 54,
            child: isImageMime(file.mime)
                ? Image.memory(
                    Uint8List.fromList(file.bytes),
                    key: Key('pending-thumb-$index'),
                    fit: BoxFit.cover,
                    gaplessPlayback: true,
                    errorBuilder: (_, __, ___) =>
                        const Icon(Icons.insert_drive_file_outlined),
                  )
                : const Icon(Icons.insert_drive_file_outlined),
          ),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              file.name,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              style: DudkaType.mono(size: 11),
            ),
          ),
          IconButton(
            key: Key('pending-remove-$index'),
            tooltip: 'Убрать',
            visualDensity: VisualDensity.compact,
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints.tightFor(width: 24, height: 24),
            onPressed: _sending
                ? null
                : () => setState(() => _pendingFiles.remove(file)),
            icon: const Icon(Icons.close, size: 16),
          ),
        ],
      ),
    );
  }

  Widget _peersList(String state, ChatSnapshot snap, {required Axis axis}) {
    if (state == 'alone' || state == 'no_network') {
      final noNetwork = state == 'no_network';
      return axis == Axis.horizontal
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SizedBox(
                  height: 24,
                  child: _participantList(snap, axis: Axis.horizontal),
                ),
                const SizedBox(height: 4),
                Row(
                  children: [
                    Text(
                      noNetwork ? 'НЕТ СЕТИ' : 'БОЛЬШЕ НИКОГО РЯДОМ',
                      key: Key(
                        noNetwork
                            ? 'chat-peers-no-network'
                            : 'chat-peers-alone',
                      ),
                    ),
                    if (!noNetwork) ...[
                      const SizedBox(width: 12),
                      OutlinedButton(
                        key: const Key('chat-seek'),
                        onPressed: _seeking ? null : _seek,
                        child: Text(_seeking ? 'ИЩЕМ…' : 'ИСКАТЬ'),
                      ),
                    ],
                    if (_seekError != null) ...[
                      const SizedBox(width: 8),
                      Flexible(
                        child: Text(
                          _seekError!,
                          style: const TextStyle(color: Colors.red),
                        ),
                      ),
                    ],
                  ],
                ),
              ],
            )
          : Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _participantTile(snap.onlineParticipants.first, snap),
                const SizedBox(height: 8),
                Text(
                  noNetwork ? 'НЕТ СЕТИ' : 'БОЛЬШЕ НИКОГО РЯДОМ',
                  key: Key(
                    noNetwork ? 'chat-peers-no-network' : 'chat-peers-alone',
                  ),
                ),
                if (!noNetwork) ...[
                  const SizedBox(height: 8),
                  OutlinedButton(
                    key: const Key('chat-seek'),
                    onPressed: _seeking ? null : _seek,
                    child: Text(_seeking ? 'ИЩЕМ…' : 'ИСКАТЬ'),
                  ),
                ],
                if (_seekError != null) ...[
                  const SizedBox(height: 4),
                  Text(_seekError!, style: const TextStyle(color: Colors.red)),
                ],
              ],
            );
    }
    return _participantList(snap, axis: axis);
  }

  Widget _participantList(ChatSnapshot snap, {required Axis axis}) {
    final participants = snap.onlineParticipants;
    if (axis == Axis.horizontal) {
      return ListView.separated(
        scrollDirection: Axis.horizontal,
        itemCount: participants.length,
        separatorBuilder: (_, __) => const SizedBox(width: 16),
        itemBuilder: (context, i) {
          return Center(child: _participantTile(participants[i], snap));
        },
      );
    }
    return ListView.builder(
      itemCount: participants.length,
      itemBuilder: (context, i) {
        return Padding(
          padding: const EdgeInsets.only(bottom: 6),
          child: _participantTile(participants[i], snap),
        );
      },
    );
  }

  Widget _participantTile(PeerInfo peer, ChatSnapshot snap) {
    final isMe =
        peer.peerId == snap.me.peerId ||
        (snap.me.peerId.trim().isEmpty && peer.peerId == 'self');
    return Text(
      isMe ? '${peer.displayName} · ВЫ' : peer.displayName,
      key: Key(isMe ? 'chat-peer-self' : 'chat-peer-${peer.peerId}'),
      style: DudkaType.mono(
        size: 13,
        weight: isMe ? FontWeight.w700 : FontWeight.w400,
      ),
    );
  }

  Widget _feedPane(ChatSnapshot snap) {
    if (snap.messages.isEmpty) {
      return const Text('—', key: Key('chat-feed-empty'));
    }
    return ListView.builder(
      itemCount: snap.messages.length,
      itemBuilder: (context, i) {
        final m = snap.messages[i];
        if (!m.isFileAnnounce) {
          return Text(
            formatFeedLine(m),
            key: Key('chat-msg-${m.msgId.isEmpty ? i : m.msgId}'),
          );
        }
        final tr = snap.transferFor(m.fileId);
        return Padding(
          key: Key('chat-msg-${m.msgId.isEmpty ? i : m.msgId}'),
          padding: const EdgeInsets.only(bottom: 8),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(formatFeedLine(m)),
              ..._thumbWidgets(m),
              const SizedBox(height: 4),
              _fileActions(m.fileId, tr),
            ],
          ),
        );
      },
    );
  }

  List<Widget> _thumbWidgets(ChatMessage m) {
    switch (feedThumbKind(m)) {
      case FeedThumbKind.image:
        final bytes = decodeThumbBytes(m)!;
        return [
          const SizedBox(height: 4),
          Image.memory(
            bytes,
            key: Key('file-thumb-${m.fileId}'),
            height: 96,
            fit: BoxFit.contain,
            gaplessPlayback: true,
          ),
        ];
      case FeedThumbKind.heicMark:
        return [
          const SizedBox(height: 4),
          Text('HEIC', key: Key('file-heic-${m.fileId}')),
        ];
      case FeedThumbKind.none:
        return const [];
    }
  }

  Widget _fileActions(String fileId, TransferInfo? tr) {
    final status = tr?.status ?? '';
    if (status == 'downloading') {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          StepProgress(percent: tr!.percent, key: Key('file-steps-$fileId')),
          const SizedBox(height: 4),
          Row(
            children: [
              Text(
                '${tr.percent}%',
                key: Key('file-progress-$fileId'),
                style: DudkaType.mono(size: 12, color: DudkaColors.segment),
              ),
              const SizedBox(width: 8),
              TextButton(
                key: Key('file-cancel-$fileId'),
                onPressed: () => _cancelFile(fileId),
                child: const Text('ОТМЕНА'),
              ),
            ],
          ),
        ],
      );
    }
    if (status == 'done') {
      final path = tr?.path.trim() ?? '';
      return Column(
        key: Key('file-done-$fileId'),
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(path.isEmpty ? 'скачано · путь не получен' : 'скачано · $path'),
          if (path.isNotEmpty)
            TextButton.icon(
              key: Key('file-reveal-$fileId'),
              onPressed: () => _revealFile(path),
              icon: const Icon(Icons.folder_open_outlined),
              label: Text(
                Platform.isMacOS ? 'ПОКАЗАТЬ В FINDER' : 'ПОКАЗАТЬ ФАЙЛ',
              ),
            ),
        ],
      );
    }
    if (status == 'cancelled') {
      return Text('отменено', key: Key('file-cancelled-$fileId'));
    }
    if (status == 'error') {
      return Text('ошибка', key: Key('file-error-$fileId'));
    }
    return TextButton(
      key: Key('file-fetch-$fileId'),
      onPressed: () => _fetchFile(fileId),
      child: const Text('СКАЧАТЬ'),
    );
  }
}
