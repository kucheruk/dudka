import 'package:dudka/engine/client.dart';

/// Live: start async fetch and optionally wait for done/cancel.
/// Usage: dart run tool/live_fetch.dart <engine-base-url> <file_id> [--wait|--cancel|--cancel-at N]
Future<void> main(List<String> args) async {
  if (args.length < 2) {
    throw StateError(
      'usage: dart run tool/live_fetch.dart <engine-base-url> <file_id> [--wait|--cancel|--cancel-at N]',
    );
  }
  final c = EngineClient(baseUrl: args.first);
  final fileId = args[1];
  var waitDone = false;
  var cancelNow = false;
  int? cancelAt;
  for (var i = 2; i < args.length; i++) {
    if (args[i] == '--wait') waitDone = true;
    if (args[i] == '--cancel') cancelNow = true;
    if (args[i] == '--cancel-at' && i + 1 < args.length) {
      cancelAt = int.tryParse(args[++i]);
    }
  }
  var tr = await c.startFetch(fileId);
  // ignore: avoid_print
  print('live_fetch started status=${tr.status} percent=${tr.percent}');
  if (cancelNow) {
    try {
      tr = await c.cancelFetch(fileId);
      // ignore: avoid_print
      print('live_fetch cancelled status=${tr.status} percent=${tr.percent}');
      c.close();
      if (tr.status != 'cancelled') {
        throw StateError('expected cancelled, got ${tr.status}');
      }
    } on EngineException catch (e) {
      // Tiny LAN downloads may finish before cancel (same as scripts/file_cancel_test.sh).
      if (e.message.contains('already done')) {
        // ignore: avoid_print
        print('live_fetch cancel_after_done_ok');
        c.close();
        return;
      }
      c.close();
      rethrow;
    }
    return;
  }
  if (cancelAt != null || waitDone) {
    for (var i = 0; i < 80; i++) {
      await Future<void>.delayed(const Duration(milliseconds: 50));
      final list = await c.fetchTransfers();
      tr = list.firstWhere(
        (t) => t.fileId == fileId,
        orElse: () => tr,
      );
      if (cancelAt != null && tr.percent >= cancelAt && tr.status == 'downloading') {
        tr = await c.cancelFetch(fileId);
        // ignore: avoid_print
        print('live_fetch cancelled status=${tr.status} percent=${tr.percent}');
        c.close();
        if (tr.status != 'cancelled') {
          throw StateError('expected cancelled, got ${tr.status}');
        }
        return;
      }
      if (tr.status == 'done' || tr.status == 'cancelled' || tr.status == 'error') {
        break;
      }
    }
  }
  // ignore: avoid_print
  print('live_fetch ok status=${tr.status} percent=${tr.percent}');
  c.close();
  if (waitDone && tr.status != 'done') {
    throw StateError('expected done, got ${tr.status}');
  }
}
