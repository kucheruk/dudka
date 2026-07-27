import 'dart:async';

import 'package:flutter/material.dart';

import '../engine/client.dart';

/// Chat wireframe: status strip + peers + text feed (P063 / DUD-UI-101).
/// Compose send lands in P064; DESIGN step-row in P069.
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
  ChatSnapshot? _snap;
  Object? _error;
  bool _loading = true;
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

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Чат')),
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
        // Keep legacy nick key for P062 tests during transition.
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
        const SizedBox(height: 8),
        const Text(
          'compose скоро (P064)',
          key: Key('chat-compose-soon'),
          style: TextStyle(color: Colors.black54),
        ),
      ],
    );
  }

  Widget _peersPane(String state, ChatSnapshot snap) {
    if (state == 'no_network') {
      return const Text('НЕТ СЕТИ', key: Key('chat-peers-no-network'));
    }
    if (state == 'alone') {
      return const Text('НИКОГО РЯДОМ', key: Key('chat-peers-alone'));
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
