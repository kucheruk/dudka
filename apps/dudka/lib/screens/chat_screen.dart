import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';

import '../engine/client.dart';
import 'settings_nick_screen.dart';

/// Chat shell: status/peers/feed/compose + files/thumbs (P063–P068).
/// DESIGN step-row lands in P069.
class ChatScreen extends StatefulWidget {
  const ChatScreen({
    super.key,
    required this.client,
    this.pollInterval = const Duration(seconds: 1),
    this.pickLocalFile,
  });

  final EngineClient client;
  final Duration pollInterval;
  final LocalFilePicker? pickLocalFile;

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _compose = TextEditingController();
  ChatSnapshot? _snap;
  Object? _error;
  String? _sendError;
  String? _seekError;
  String? _fileError;
  bool _loading = true;
  bool _sending = false;
  bool _seeking = false;
  bool _announcing = false;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _refresh();
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
    if (text.isEmpty) return;
    setState(() {
      _sending = true;
      _sendError = null;
    });
    try {
      await widget.client.sendText(text);
      _compose.clear();
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
        builder: (_) => SettingsNickScreen(
          client: widget.client,
          initialNick: current,
        ),
      ),
    );
    if (!mounted) return;
    if (updated != null) {
      await _refresh();
    }
  }

  Future<LocalFileBytes?> _defaultPickPath() async {
    final ctrl = TextEditingController();
    final path = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Файл'),
        content: TextField(
          controller: ctrl,
          decoration: const InputDecoration(hintText: '/path/to/file'),
          autofocus: true,
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Отмена')),
          TextButton(
            onPressed: () => Navigator.pop(ctx, ctrl.text.trim()),
            child: const Text('OK'),
          ),
        ],
      ),
    );
    ctrl.dispose();
    if (path == null || path.isEmpty) return null;
    final f = File(path);
    if (!f.existsSync()) {
      throw EngineException('файл не найден: $path');
    }
    final name = path.split(Platform.pathSeparator).last;
    final mime = name.endsWith('.txt') ? 'text/plain' : 'application/octet-stream';
    return LocalFileBytes(name: name, mime: mime, bytes: await f.readAsBytes());
  }

  Future<void> _announcePicked() async {
    if (_announcing) return;
    setState(() {
      _announcing = true;
      _fileError = null;
    });
    try {
      final picker = widget.pickLocalFile ?? _defaultPickPath;
      final picked = await picker();
      if (picked == null) return;
      await widget.client.announceFile(
        name: picked.name,
        mime: picked.mime,
        content: picked.bytes,
      );
      await _refresh();
    } catch (e) {
      if (!mounted) return;
      setState(() => _fileError = e.toString());
    } finally {
      if (mounted) setState(() => _announcing = false);
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Чат'),
        actions: [
          IconButton(
            key: const Key('chat-settings'),
            tooltip: 'Настройки',
            icon: const Icon(Icons.settings_outlined),
            onPressed: _openSettings,
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: _body(),
      ),
    );
  }

  Widget _body() {
    if (_loading && _snap == null) {
      return const Center(child: CircularProgressIndicator(key: Key('chat-loading')));
    }
    if (_error != null && _snap == null) {
      return Text('engine offline\n$_error', key: const Key('chat-error'));
    }
    final snap = _snap!;
    final state = chatNetworkState(network: snap.network, peerCount: snap.peers.length);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          formatStatusStrip(snap),
          key: const Key('chat-status'),
          style: const TextStyle(fontWeight: FontWeight.bold),
        ),
        Text('вы: ${snap.me.name}', key: const Key('chat-nick'), style: const TextStyle(color: Colors.black54)),
        const SizedBox(height: 12),
        const Text('PEERS', style: TextStyle(fontWeight: FontWeight.bold, letterSpacing: 1.2)),
        const SizedBox(height: 4),
        Expanded(
          flex: 2,
          child: Container(
            key: const Key('chat-peers'),
            alignment: Alignment.topLeft,
            child: _peersPane(state, snap),
          ),
        ),
        const SizedBox(height: 8),
        const Text('ЛЕНТА', style: TextStyle(fontWeight: FontWeight.bold, letterSpacing: 1.2)),
        const SizedBox(height: 4),
        Expanded(
          flex: 5,
          child: Container(
            key: const Key('chat-feed'),
            alignment: Alignment.topLeft,
            child: _feedPane(snap),
          ),
        ),
        if (_sendError != null) Text(_sendError!, style: const TextStyle(color: Colors.red)),
        if (_fileError != null) Text(_fileError!, style: const TextStyle(color: Colors.red)),
        const SizedBox(height: 8),
        Row(
          children: [
            Expanded(
              child: TextField(
                key: const Key('chat-compose'),
                controller: _compose,
                enabled: !_sending,
                decoration: const InputDecoration(
                  hintText: 'текст…',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
                textInputAction: TextInputAction.send,
                onSubmitted: (_) => _blow(),
              ),
            ),
            const SizedBox(width: 8),
            OutlinedButton(
              key: const Key('chat-file'),
              onPressed: _announcing ? null : _announcePicked,
              child: Text(_announcing ? '…' : 'ФАЙЛ'),
            ),
            const SizedBox(width: 8),
            FilledButton(
              key: const Key('chat-blow'),
              onPressed: _sending ? null : _blow,
              child: const Text('ДУНУТЬ'),
            ),
          ],
        ),
      ],
    );
  }

  Widget _peersPane(String state, ChatSnapshot snap) {
    if (state == 'no_network') {
      return const Text('НЕТ СЕТИ', key: Key('chat-peers-no-network'));
    }
    if (state == 'alone') {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('НИКОГО РЯДОМ', key: Key('chat-peers-alone')),
          const SizedBox(height: 8),
          OutlinedButton(
            key: const Key('chat-seek'),
            onPressed: _seeking ? null : _seek,
            child: Text(_seeking ? 'ИЩЕМ…' : 'ИСКАТЬ'),
          ),
          if (_seekError != null) ...[
            const SizedBox(height: 4),
            Text(_seekError!, style: const TextStyle(color: Colors.red)),
          ],
        ],
      );
    }
    return ListView.builder(
      itemCount: snap.peers.length,
      itemBuilder: (context, i) {
        final p = snap.peers[i];
        return Text(p.displayName, key: Key('chat-peer-${p.peerId}'));
      },
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
      return Row(
        children: [
          Text('${tr!.percent}%', key: Key('file-progress-$fileId')),
          const SizedBox(width: 8),
          TextButton(
            key: Key('file-cancel-$fileId'),
            onPressed: () => _cancelFile(fileId),
            child: const Text('ОТМЕНА'),
          ),
        ],
      );
    }
    if (status == 'done') {
      return Text('готово', key: Key('file-done-$fileId'));
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
