import 'package:flutter/material.dart';

import '../engine/client.dart';

/// Minimal chat shell after first-run (P062). Full wireframe arrives in P063.
class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key, required this.client});

  final EngineClient client;

  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  late Future<MeInfo> _me;

  @override
  void initState() {
    super.initState();
    _me = widget.client.fetchMe();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Чат')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: FutureBuilder<MeInfo>(
          future: _me,
          builder: (context, snap) {
            if (snap.connectionState != ConnectionState.done) {
              return const Center(child: CircularProgressIndicator());
            }
            if (snap.hasError) {
              return Text('engine offline\n${snap.error}', key: const Key('chat-error'));
            }
            final me = snap.data!;
            return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('вы: ${me.name}', key: const Key('chat-nick')),
                const SizedBox(height: 16),
                const Text('ЛЕНТА', style: TextStyle(fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                const Text('—', key: Key('chat-feed-empty')),
                const Spacer(),
                const Text(
                  'compose скоро (P063/P064)',
                  style: TextStyle(color: Colors.black54),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}
