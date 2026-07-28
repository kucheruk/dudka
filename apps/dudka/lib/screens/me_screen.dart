import 'package:flutter/material.dart';

import '../engine/client.dart';

/// Skeleton home: shows dudkad `GET /me` (P061).
class MeScreen extends StatefulWidget {
  const MeScreen({super.key, required this.engineBase, this.client});

  final String engineBase;
  final EngineClient? client;

  @override
  State<MeScreen> createState() => _MeScreenState();
}

class _MeScreenState extends State<MeScreen> {
  late final EngineClient _client;
  late Future<MeInfo> _me;

  @override
  void initState() {
    super.initState();
    _client = widget.client ?? EngineClient(baseUrl: widget.engineBase);
    _me = _client.fetchMe();
  }

  @override
  void dispose() {
    if (widget.client == null) {
      _client.close();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('ДУДКА')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: FutureBuilder<MeInfo>(
          future: _me,
          builder: (context, snap) {
            if (snap.connectionState != ConnectionState.done) {
              return const Center(child: CircularProgressIndicator());
            }
            if (snap.hasError) {
              return Text(
                'движок недоступен /me\n${snap.error}',
                key: const Key('me-error'),
              );
            }
            final me = snap.data!;
            return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('GET /me',
                    style:
                        TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                const SizedBox(height: 12),
                Text('name: ${me.name}', key: const Key('me-name')),
                Text('peer_id: ${me.peerId}', key: const Key('me-peer-id')),
                const SizedBox(height: 24),
                Text('engine: ${widget.engineBase}',
                    style: const TextStyle(color: Colors.black54)),
              ],
            );
          },
        ),
      ),
    );
  }
}
