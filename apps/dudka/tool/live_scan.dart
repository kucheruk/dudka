import 'package:dudka/engine/client.dart';

/// Live: EngineClient.startScan against a running dudkad.
/// Usage: dart run tool/live_scan.dart <engine-base-url> [host] [port]
Future<void> main(List<String> args) async {
  if (args.isEmpty) {
    throw StateError('usage: dart run tool/live_scan.dart <engine-base-url> [host] [port]');
  }
  final c = EngineClient(baseUrl: args.first);
  List<String>? hosts;
  int? port;
  if (args.length >= 2 && args[1].trim().isNotEmpty) {
    hosts = [args[1].trim()];
  }
  if (args.length >= 3) {
    port = int.tryParse(args[2]);
  }
  final res = await c.startScan(hosts: hosts, port: port);
  if (hosts != null && res.found < 1) {
    throw StateError('scan found=0 probed=${res.probed}');
  }
  // ignore: avoid_print
  print('live_scan ok found=${res.found} probed=${res.probed}');
  c.close();
}
