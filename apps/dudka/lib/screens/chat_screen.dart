import 'dart:async';

import 'package:flutter/material.dart';

import '../engine/client.dart';
import 'settings_nick_screen.dart';

/// Chat wireframe: status/peers/feed/compose + alone «ИСКАТЬ» + nick settings (P063–P066).
/// DESIGN step-row lands in P069.
class ChatScreen extends StatefulWidget {
  const ChatScreen({
    super.key,
    required this.client,
    this.pollInterval = const Duration(seconds: 1),
  });

  final EngineClient client;
  final Duration pollInterval;

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _compose = TextEditingController();
  ChatSnapshot? _snap;
  Object? _error;
  String? _sendError;
  String? _seekError;
  bool _loading = true;
  bool _sending = false;
  bool _seeking = false;
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
        if (_sendError != null) ...[
          const SizedBox(height: 4),
          Text(_sendError!, style: const TextStyle(color: Colors.red)),
        ],
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
        return Text(
          formatFeedLine(m),
          key: Key('chat-msg-${m.msgId.isEmpty ? i : m.msgId}'),
        );
      },
    );
  }
}
