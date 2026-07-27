import 'package:dudka/engine/client.dart';

/// Live check: EngineClient.fetchMe against a running dudkad.
/// Usage: dart run tool/live_me.dart http://127.0.0.1:PORT
Future<void> main(List<String> args) async {
  if (args.isEmpty) {
    throw StateError('usage: dart run tool/live_me.dart <engine-base-url>');
  }
  final c = EngineClient(baseUrl: args.first);
  final me = await c.fetchMe();
  if (me.peerId.isEmpty) {
    throw StateError('empty peer_id');
  }
  // ignore: avoid_print
  print('live_me ok name=${me.name} peer_id=${me.peerId}');
  c.close();
}
