import 'dart:io';

import 'package:dudka/engine/client.dart';

/// Live: announce a local file via EngineClient.
/// Usage: dart run tool/live_announce.dart <engine-base-url> <path> [mime]
Future<void> main(List<String> args) async {
  if (args.length < 2) {
    throw StateError('usage: dart run tool/live_announce.dart <engine-base-url> <path> [mime]');
  }
  final path = args[1];
  final f = File(path);
  if (!f.existsSync()) throw StateError('missing file $path');
  final name = path.split(Platform.pathSeparator).last;
  final mime = args.length >= 3 ? args[2] : 'application/octet-stream';
  final c = EngineClient(baseUrl: args.first);
  final msg = await c.announceFile(name: name, mime: mime, content: await f.readAsBytes());
  // ignore: avoid_print
  print('live_announce ok file_id=${msg.fileId} name=${msg.fileName}');
  c.close();
}
