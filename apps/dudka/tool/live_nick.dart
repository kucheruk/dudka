import 'package:dudka/engine/client.dart';

/// Live: EngineClient.setNick against a running dudkad.
/// Usage: dart run tool/live_nick.dart <engine-base-url> <nick>
Future<void> main(List<String> args) async {
  if (args.length < 2) {
    throw StateError('usage: dart run tool/live_nick.dart <engine-base-url> <nick>');
  }
  final c = EngineClient(baseUrl: args.first);
  final nick = args.sublist(1).join(' ');
  final me = await c.setNick(nick);
  if (me.name != nick) {
    throw StateError('expected name=$nick got=${me.name}');
  }
  // ignore: avoid_print
  print('live_nick ok name=${me.name}');
  c.close();
}
