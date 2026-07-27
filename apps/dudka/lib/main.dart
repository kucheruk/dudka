import 'package:flutter/material.dart';

import 'engine.dart';

/// Spike entry (P060): show GET /me from dudkad loopback.
///
/// Pass engine base via `--dart-define=DUDKA_ENGINE=http://127.0.0.1:PORT`
/// or rely on default `http://127.0.0.1:17880`.
void main() {
  const engine = String.fromEnvironment(
    'DUDKA_ENGINE',
    defaultValue: 'http://127.0.0.1:17880',
  );
  runApp(DudkaSpikeApp(engineBase: engine));
}

class DudkaSpikeApp extends StatelessWidget {
  const DudkaSpikeApp({super.key, required this.engineBase});

  final String engineBase;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'ДУДКА',
      debugShowCheckedModeBanner: false,
      home: MeHelloScreen(engineBase: engineBase),
    );
  }
}

class MeHelloScreen extends StatefulWidget {
  const MeHelloScreen({super.key, required this.engineBase, this.client});

  final String engineBase;
  final EngineClient? client;

  @override
  State<MeHelloScreen> createState() => _MeHelloScreenState();
}

class _MeHelloScreenState extends State<MeHelloScreen> {
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
      appBar: AppBar(title: const Text('ДУДКА · spike')),
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
                'engine offline /me\n${snap.error}',
                key: const Key('me-error'),
              );
            }
            final me = snap.data!;
            return Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('GET /me', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                const SizedBox(height: 12),
                Text('name: ${me.name}', key: const Key('me-name')),
                Text('peer_id: ${me.peerId}', key: const Key('me-peer-id')),
                const SizedBox(height: 24),
                Text('engine: ${widget.engineBase}', style: const TextStyle(color: Colors.black54)),
              ],
            );
          },
        ),
      ),
    );
  }
}
