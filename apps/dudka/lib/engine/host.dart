import 'dart:async';
import 'dart:convert';
import 'dart:io';

/// Spawns local `dudkad` (subprocess + loopback) and yields its HTTP base URL (P061).
class EngineHost {
  EngineHost({required this.binaryPath, required this.dataDir, this.name = ''});

  final String binaryPath;
  final String dataDir;
  final String name;

  Process? _proc;
  String? _baseUrl;

  String? get baseUrl => _baseUrl;
  bool get isRunning => _proc != null;

  static String? parseListenLine(String line) {
    final t = line.trim();
    const prefix = 'listen=';
    if (!t.startsWith(prefix)) return null;
    final addr = t.substring(prefix.length).trim();
    return addr.isEmpty ? null : addr;
  }

  static bool parseReadyLine(String line) => line.trim().startsWith('ready ');

  static String baseUrlFromListen(String listen) => 'http://$listen';

  /// Start dudkad with ephemeral loopback listen; wait until ready + listen=.
  Future<String> start({Duration timeout = const Duration(seconds: 8)}) async {
    if (_baseUrl != null) return _baseUrl!;
    final bin = File(binaryPath);
    if (!bin.existsSync()) {
      throw EngineHostException('dudkad binary missing: $binaryPath');
    }
    await Directory(dataDir).create(recursive: true);

    final proc = await Process.start(
      binaryPath,
      arguments(),
      mode: ProcessStartMode.normal,
    );
    _proc = proc;

    String? listen;
    var ready = false;
    final done = Completer<void>();
    // Keep draining stdout/stderr so dudkad cannot block on a full pipe.
    proc.stdout.transform(utf8.decoder).transform(const LineSplitter()).listen((
      line,
    ) {
      listen ??= parseListenLine(line);
      if (parseReadyLine(line)) ready = true;
      if (listen != null && ready && !done.isCompleted) {
        done.complete();
      }
    });
    proc.stderr.transform(utf8.decoder).listen((_) {});

    try {
      await done.future.timeout(timeout);
    } on TimeoutException {
      await stop();
      throw EngineHostException('dudkad did not become ready in $timeout');
    }

    _baseUrl = baseUrlFromListen(listen!);
    return _baseUrl!;
  }

  List<String> arguments() => [
    '-data-dir',
    dataDir,
    if (name.trim().isNotEmpty) ...['-name', name.trim()],
    '-listen',
    '127.0.0.1:0',
  ];

  Future<void> stop() async {
    final p = _proc;
    _proc = null;
    _baseUrl = null;
    if (p == null) return;
    p.kill();
    try {
      await p.exitCode.timeout(const Duration(seconds: 2));
    } catch (_) {}
  }
}

class EngineHostException implements Exception {
  EngineHostException(this.message);
  final String message;

  @override
  String toString() => message;
}
