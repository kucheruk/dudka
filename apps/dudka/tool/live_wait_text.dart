import 'package:dudka/engine/client.dart';

/// Poll EngineClient.fetchSnapshot until [needle] appears in feed (P071).
/// Usage: dart run tool/live_wait_text.dart <engine-base-url> <needle...> [--timeout-ms N]
Future<void> main(List<String> args) async {
  if (args.length < 2) {
    throw StateError(
      'usage: dart run tool/live_wait_text.dart <engine-base-url> <needle...> [--timeout-ms N]',
    );
  }
  final base = args.first;
  var timeoutMs = 8000;
  final needleParts = <String>[];
  for (var i = 1; i < args.length; i++) {
    if (args[i] == '--timeout-ms' && i + 1 < args.length) {
      timeoutMs = int.parse(args[++i]);
      continue;
    }
    needleParts.add(args[i]);
  }
  final needle = needleParts.join(' ').trim();
  if (needle.isEmpty) throw StateError('needle is required');

  final c = EngineClient(baseUrl: base);
  final deadline = DateTime.now().add(Duration(milliseconds: timeoutMs));
  while (DateTime.now().isBefore(deadline)) {
    final snap = await c.fetchSnapshot();
    for (final m in snap.messages) {
      final hay = [
        m.text,
        m.fileId,
        m.fileName,
        m.displayNameAtSend,
        m.msgId,
      ].join(' ');
      if (hay.contains(needle)) {
        // ignore: avoid_print
        print('live_wait_text ok needle=$needle msgs=${snap.messages.length}');
        c.close();
        return;
      }
    }
    await Future<void>.delayed(const Duration(milliseconds: 100));
  }
  c.close();
  throw StateError('timeout waiting for needle="$needle" on $base');
}
