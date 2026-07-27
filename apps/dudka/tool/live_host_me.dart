import 'dart:io';

import 'package:dudka/engine/client.dart';
import 'package:dudka/engine/host.dart';

/// Spawn dudkad via EngineHost and call GET /me (P061 proof).
/// Usage: dart run tool/live_host_me.dart /path/to/dudkad
Future<void> main(List<String> args) async {
  if (args.isEmpty) {
    throw StateError('usage: dart run tool/live_host_me.dart <dudkad-bin>');
  }
  final dataDir = Directory('${Directory.systemTemp.path}/dudka-host-me-${pid}');
  final host = EngineHost(binaryPath: args.first, dataDir: dataDir.path, name: 'Skeleton');
  try {
    final base = await host.start();
    final client = EngineClient(baseUrl: base);
    final me = await client.fetchMe();
    if (me.name != 'Skeleton') {
      throw StateError('unexpected name=${me.name}');
    }
    if (me.peerId.isEmpty) {
      throw StateError('empty peer_id');
    }
    // ignore: avoid_print
    print('live_host_me ok base=$base name=${me.name} peer_id=${me.peerId}');
    client.close();
  } finally {
    await host.stop();
  }
}
