import 'package:dudka/engine/client.dart';

/// Live: EngineClient.sendText against a running dudkad.
/// Usage: dart run tool/live_send.dart <engine-base-url> <text>
Future<void> main(List<String> args) async {
  if (args.length < 2) {
    throw StateError('usage: dart run tool/live_send.dart <engine-base-url> <text>');
  }
  final c = EngineClient(baseUrl: args.first);
  final text = args.sublist(1).join(' ');
  final res = await c.sendText(text);
  // ignore: avoid_print
  print('live_send ok status=${res.status} text=${res.text}');
  c.close();
}
