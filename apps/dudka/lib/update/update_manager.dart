import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:package_info_plus/package_info_plus.dart';

const updateManifestUrl = 'https://zamoo.team/dudka/update.json';
const maxDesktopUpdateBytes = 1024 * 1024 * 1024;

enum UpdatePhase {
  idle,
  checking,
  downloading,
  ready,
  activating,
  unavailable,
  failed,
}

@immutable
class UpdateSnapshot {
  const UpdateSnapshot({
    this.phase = UpdatePhase.idle,
    this.version,
    this.error,
  });

  final UpdatePhase phase;
  final String? version;
  final String? error;

  bool get isReady => phase == UpdatePhase.ready;
}

@immutable
class UpdatePackage {
  const UpdatePackage({
    required this.url,
    required this.sha256Hex,
    required this.size,
    required this.format,
  });

  final Uri url;
  final String sha256Hex;
  final int size;
  final String format;

  factory UpdatePackage.fromJson(Object? value) {
    if (value is! Map<String, dynamic>) {
      throw const FormatException('update package must be an object');
    }
    final rawUrl = value['url'];
    final rawHash = value['sha256'];
    final rawSize = value['size'];
    final rawFormat = value['format'];
    if (rawUrl is! String ||
        rawHash is! String ||
        rawSize is! int ||
        rawFormat is! String) {
      throw const FormatException('update package fields are invalid');
    }
    final url = Uri.tryParse(rawUrl);
    if (url == null || url.scheme != 'https' || url.host.isEmpty) {
      throw const FormatException('update package URL must be HTTPS');
    }
    final hash = rawHash.toLowerCase();
    if (!RegExp(r'^[0-9a-f]{64}$').hasMatch(hash)) {
      throw const FormatException('update package sha256 is invalid');
    }
    if (rawSize <= 0 || rawSize > maxDesktopUpdateBytes) {
      throw const FormatException('update package size is outside limits');
    }
    if (rawFormat != 'zip') {
      throw const FormatException('only zip desktop updates are supported');
    }
    return UpdatePackage(
      url: url,
      sha256Hex: hash,
      size: rawSize,
      format: rawFormat,
    );
  }
}

@immutable
class UpdateManifest {
  const UpdateManifest({
    required this.version,
    required this.packages,
  });

  final String version;
  final Map<String, UpdatePackage> packages;

  factory UpdateManifest.fromJson(Object? value) {
    if (value is! Map<String, dynamic> || value['schema'] != 1) {
      throw const FormatException('unsupported update manifest schema');
    }
    final version = value['version'];
    final rawPackages = value['packages'];
    if (version is! String || !isSemanticVersion(version)) {
      throw const FormatException('update version must be SemVer');
    }
    if (rawPackages is! Map<String, dynamic>) {
      throw const FormatException('update packages must be an object');
    }
    final packages = <String, UpdatePackage>{};
    for (final entry in rawPackages.entries) {
      packages[entry.key] = UpdatePackage.fromJson(entry.value);
    }
    return UpdateManifest(version: version, packages: packages);
  }

  factory UpdateManifest.parse(String source) {
    return UpdateManifest.fromJson(jsonDecode(source));
  }
}

bool isSemanticVersion(String value) {
  return RegExp(r'^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$').hasMatch(value);
}

bool isVersionNewer(String candidate, String current) {
  if (!isSemanticVersion(candidate) || !isSemanticVersion(current)) {
    return false;
  }
  final a = candidate.split('.').map(int.parse).toList(growable: false);
  final b = current.split('.').map(int.parse).toList(growable: false);
  for (var i = 0; i < 3; i++) {
    if (a[i] != b[i]) return a[i] > b[i];
  }
  return false;
}

String? currentUpdatePlatform() {
  if (Platform.isMacOS) return 'macos-universal';
  if (Platform.isWindows) return 'windows-amd64';
  return null;
}

typedef ExitApplication = void Function(int code);
typedef BeforeUpdateExit = Future<void> Function();

void _exitApplication(int code) => exit(code);

abstract class UpdateActivator {
  Future<void> schedule({
    required UpdatePackage package,
    required File archive,
    required String version,
    required Duration delay,
  });
}

abstract class UpdateController implements Listenable {
  UpdateSnapshot get snapshot;
  void start();
  Future<void> activate();
}

class UpdateManager extends ChangeNotifier implements UpdateController {
  UpdateManager({
    required this.currentVersion,
    required this.platformKey,
    required this.cacheRoot,
    required this.activator,
    required this.beforeExit,
    required this.exitApplication,
    http.Client? client,
    Uri? manifestUri,
    this.checkInterval = const Duration(minutes: 15),
  })  : _client = client ?? http.Client(),
        _ownsClient = client == null,
        manifestUri = manifestUri ?? Uri.parse(updateManifestUrl);

  final String currentVersion;
  final String platformKey;
  final Directory cacheRoot;
  final UpdateActivator activator;
  final BeforeUpdateExit beforeExit;
  final ExitApplication exitApplication;
  final Uri manifestUri;
  final Duration checkInterval;
  final http.Client _client;
  final bool _ownsClient;

  UpdateSnapshot _snapshot = const UpdateSnapshot();

  @override
  UpdateSnapshot get snapshot => _snapshot;

  Timer? _timer;
  File? _readyArchive;
  UpdatePackage? _readyPackage;
  bool _started = false;
  bool _busy = false;

  static Future<UpdateManager?> forCurrentPlatform({
    required BeforeUpdateExit beforeExit,
    ExitApplication exitApplication = _exitApplication,
  }) async {
    final platform = currentUpdatePlatform();
    if (platform == null) return null;
    final packageInfo = await PackageInfo.fromPlatform();
    return UpdateManager(
      currentVersion: packageInfo.version,
      platformKey: platform,
      cacheRoot: Directory(
        '${Directory.systemTemp.path}${Platform.pathSeparator}dudka-updates',
      ),
      activator: PlatformUpdateActivator(),
      beforeExit: beforeExit,
      exitApplication: exitApplication,
    );
  }

  @override
  void start() {
    if (_started) return;
    _started = true;
    unawaited(checkAndStage());
    if (checkInterval > Duration.zero) {
      _timer = Timer.periodic(checkInterval, (_) => checkAndStage());
    }
  }

  Future<void> checkAndStage() async {
    if (_busy || _snapshot.phase == UpdatePhase.activating) return;
    _busy = true;
    final previousReady = _snapshot.isReady ? _snapshot : null;
    _setSnapshot(const UpdateSnapshot(phase: UpdatePhase.checking));
    try {
      final response = await _client.get(manifestUri, headers: const {
        'Cache-Control': 'no-cache'
      }).timeout(const Duration(seconds: 10));
      if (response.statusCode != HttpStatus.ok) {
        throw HttpException('update manifest HTTP ${response.statusCode}');
      }
      final manifest = UpdateManifest.parse(response.body);
      if (!isVersionNewer(manifest.version, currentVersion)) {
        _clearReady();
        _setSnapshot(const UpdateSnapshot(phase: UpdatePhase.unavailable));
        return;
      }
      final package = manifest.packages[platformKey];
      if (package == null) {
        _clearReady();
        _setSnapshot(const UpdateSnapshot(phase: UpdatePhase.unavailable));
        return;
      }
      _setSnapshot(UpdateSnapshot(
        phase: UpdatePhase.downloading,
        version: manifest.version,
      ));
      final archive = await _downloadAndVerify(package, manifest.version);
      _readyArchive = archive;
      _readyPackage = package;
      _setSnapshot(UpdateSnapshot(
        phase: UpdatePhase.ready,
        version: manifest.version,
      ));
    } catch (error) {
      if (previousReady != null &&
          _readyArchive != null &&
          _readyPackage != null) {
        _setSnapshot(previousReady);
      } else {
        _clearReady();
        _setSnapshot(UpdateSnapshot(
          phase: UpdatePhase.failed,
          error: error.toString(),
        ));
      }
    } finally {
      _busy = false;
    }
  }

  Future<File> _downloadAndVerify(
    UpdatePackage package,
    String version,
  ) async {
    final versionDir = Directory(
      '${cacheRoot.path}${Platform.pathSeparator}$version',
    );
    await versionDir.create(recursive: true);
    final archive = File(
      '${versionDir.path}${Platform.pathSeparator}$platformKey.zip',
    );
    if (await archive.exists() && await _verifyFile(archive, package)) {
      return archive;
    }

    final partial = File('${archive.path}.partial');
    if (await partial.exists()) await partial.delete();
    final request = http.Request('GET', package.url);
    final response =
        await _client.send(request).timeout(const Duration(seconds: 10));
    if (response.statusCode != HttpStatus.ok) {
      throw HttpException('update artifact HTTP ${response.statusCode}');
    }

    final digestSink = _DigestSink();
    final hashInput = sha256.startChunkedConversion(digestSink);
    final fileOutput = partial.openWrite();
    var fileOutputClosed = false;
    var hashInputClosed = false;
    var received = 0;
    try {
      await for (final chunk
          in response.stream.timeout(const Duration(seconds: 60))) {
        received += chunk.length;
        if (received > package.size) {
          throw const FormatException('update artifact exceeds declared size');
        }
        hashInput.add(chunk);
        fileOutput.add(chunk);
      }
      await fileOutput.close();
      fileOutputClosed = true;
      hashInput.close();
      hashInputClosed = true;
    } catch (_) {
      if (!fileOutputClosed) await fileOutput.close();
      if (!hashInputClosed) hashInput.close();
      if (await partial.exists()) await partial.delete();
      rethrow;
    }

    if (received != package.size ||
        digestSink.value?.toString() != package.sha256Hex) {
      if (await partial.exists()) await partial.delete();
      throw const FormatException('update artifact integrity mismatch');
    }
    if (await archive.exists()) await archive.delete();
    return partial.rename(archive.path);
  }

  Future<bool> _verifyFile(File file, UpdatePackage package) async {
    if (await file.length() != package.size) return false;
    final digest = await sha256.bind(file.openRead()).first;
    return digest.toString() == package.sha256Hex;
  }

  @override
  Future<void> activate() async {
    final archive = _readyArchive;
    final package = _readyPackage;
    final version = _snapshot.version;
    if (archive == null ||
        package == null ||
        version == null ||
        !_snapshot.isReady) {
      throw StateError('update is not ready');
    }
    _setSnapshot(UpdateSnapshot(
      phase: UpdatePhase.activating,
      version: version,
    ));
    try {
      await activator.schedule(
        package: package,
        archive: archive,
        version: version,
        delay: const Duration(seconds: 10),
      );
      await beforeExit();
      exitApplication(0);
    } catch (error) {
      _setSnapshot(UpdateSnapshot(
        phase: UpdatePhase.failed,
        version: version,
        error: error.toString(),
      ));
      rethrow;
    }
  }

  void _clearReady() {
    _readyArchive = null;
    _readyPackage = null;
  }

  void _setSnapshot(UpdateSnapshot value) {
    _snapshot = value;
    notifyListeners();
  }

  @override
  void dispose() {
    _timer?.cancel();
    if (_ownsClient) _client.close();
    super.dispose();
  }
}

class _DigestSink implements Sink<Digest> {
  Digest? value;

  @override
  void add(Digest data) => value = data;

  @override
  void close() {}
}

class PlatformUpdateActivator implements UpdateActivator {
  PlatformUpdateActivator({
    String? executablePath,
    int? processId,
  })  : executablePath = executablePath ?? Platform.resolvedExecutable,
        processId = processId ?? pid;

  final String executablePath;
  final int processId;

  @override
  Future<void> schedule({
    required UpdatePackage package,
    required File archive,
    required String version,
    required Duration delay,
  }) async {
    if (Platform.isMacOS) {
      final bundle = macBundleFromExecutable(executablePath);
      if (bundle.path.startsWith('/Volumes/')) {
        throw StateError(
          'Сначала перенесите Дудку из DMG в Applications',
        );
      }
      await ensureDirectoryWritable(
        bundle.parent,
        probeSuffix: '$processId',
        errorTarget: bundle.path,
      );
      final script = File('${archive.parent.path}/activate-macos-$version.sh');
      await script.writeAsString(buildMacActivationScript(
        archivePath: archive.path,
        targetBundlePath: bundle.path,
        processId: processId,
        delay: delay,
        version: version,
      ));
      await Process.start(
        '/bin/sh',
        [script.path],
        mode: ProcessStartMode.detached,
      );
      return;
    }
    if (Platform.isWindows) {
      final targetExecutable = File(executablePath);
      final writeProbe = File(
        '${targetExecutable.parent.path}${Platform.pathSeparator}'
        '.dudka-update-write-test-$processId',
      );
      try {
        await writeProbe.create(exclusive: true);
        await writeProbe.delete();
      } catch (_) {
        if (await writeProbe.exists()) await writeProbe.delete();
        throw StateError(
          'Нет прав на обновление ${targetExecutable.parent.path}',
        );
      }
      final script = File(
        '${archive.parent.path}${Platform.pathSeparator}'
        'activate-windows-$version.ps1',
      );
      await script.writeAsString(buildWindowsActivationScript(
        archivePath: archive.path,
        targetExecutablePath: targetExecutable.path,
        processId: processId,
        delay: delay,
        version: version,
      ));
      await Process.start(
        'powershell.exe',
        [
          '-NoProfile',
          '-ExecutionPolicy',
          'Bypass',
          '-File',
          script.path,
        ],
        mode: ProcessStartMode.detached,
      );
      return;
    }
    throw UnsupportedError('auto-update is desktop-only');
  }
}

Future<void> ensureDirectoryWritable(
  Directory directory, {
  required String probeSuffix,
  required String errorTarget,
}) async {
  final probe = File(
    '${directory.path}${Platform.pathSeparator}'
    '.dudka-update-write-test-$probeSuffix',
  );
  try {
    await probe.create(exclusive: true);
    await probe.delete();
  } catch (_) {
    if (await probe.exists()) await probe.delete();
    throw StateError('Нет прав на обновление $errorTarget');
  }
}

Directory macBundleFromExecutable(String executablePath) {
  final executable = File(executablePath);
  final bundle = executable.parent.parent.parent;
  if (!bundle.path.toLowerCase().endsWith('.app')) {
    throw StateError('Дудка запущена не из app bundle');
  }
  return bundle;
}

String buildMacActivationScript({
  required String archivePath,
  required String targetBundlePath,
  required int processId,
  required Duration delay,
  required String version,
}) {
  final archive = _shellQuote(archivePath);
  final target = _shellQuote(targetBundlePath);
  final safeVersion = version.replaceAll(RegExp(r'[^0-9.]'), '');
  return '''#!/bin/sh
set -eu
sleep ${delay.inSeconds}
tries=0
while kill -0 $processId 2>/dev/null; do
  sleep 0.2
  tries=\$((tries + 1))
  [ "\$tries" -lt 1500 ] || exit 1
done
work="\$(mktemp -d "\${TMPDIR:-/tmp}/dudka-update.XXXXXX")"
target=$target
incoming="\${target}.incoming-$safeVersion"
backup="\${target}.backup-$safeVersion"
moved=0
cleanup() {
  status=\$?
  if [ "\$status" -ne 0 ] && [ "\$moved" -eq 1 ] && [ -d "\$backup" ]; then
    rm -rf "\$target"
    mv "\$backup" "\$target"
    /usr/bin/open "\$target" >/dev/null 2>&1 || true
  fi
  rm -rf "\$work" "\$incoming"
  trap - EXIT
  exit "\$status"
}
trap cleanup EXIT
/usr/bin/ditto -x -k $archive "\$work"
new_app="\$work/dudka.app"
[ -d "\$new_app" ] || exit 1
rm -rf "\$incoming" "\$backup"
/usr/bin/ditto "\$new_app" "\$incoming"
mv "\$target" "\$backup"
moved=1
mv "\$incoming" "\$target"
/usr/bin/open "\$target"
sleep 2
rm -rf "\$backup"
moved=0
''';
}

String buildWindowsActivationScript({
  required String archivePath,
  required String targetExecutablePath,
  required int processId,
  required Duration delay,
  required String version,
}) {
  final archive = _powerShellQuote(archivePath);
  final targetExecutable = _powerShellQuote(targetExecutablePath);
  final safeVersion = version.replaceAll(RegExp(r'[^0-9.]'), '');
  final backup = _powerShellQuote('dudka-backup-$safeVersion');
  return '''\$ErrorActionPreference = 'Stop'
Start-Sleep -Seconds ${delay.inSeconds}
Wait-Process -Id $processId -ErrorAction SilentlyContinue
\$archive = $archive
\$targetExe = $targetExecutable
\$targetDir = Split-Path -Parent \$targetExe
\$parentDir = Split-Path -Parent \$targetDir
\$work = Join-Path \$env:TEMP ('dudka-update-' + [guid]::NewGuid())
\$backup = Join-Path \$parentDir $backup
Expand-Archive -LiteralPath \$archive -DestinationPath \$work -Force
\$newExe = Get-ChildItem -Path \$work -Filter 'dudka.exe' -Recurse | Select-Object -First 1
if (\$null -eq \$newExe) { throw 'dudka.exe missing in update package' }
\$newDir = Split-Path -Parent \$newExe.FullName
if (Test-Path \$backup) { Remove-Item \$backup -Recurse -Force }
Move-Item -LiteralPath \$targetDir -Destination \$backup
try {
  Move-Item -LiteralPath \$newDir -Destination \$targetDir
  Start-Process -FilePath \$targetExe
  Start-Sleep -Seconds 2
  Remove-Item \$backup -Recurse -Force
} catch {
  if (Test-Path \$targetDir) { Remove-Item \$targetDir -Recurse -Force }
  Move-Item -LiteralPath \$backup -Destination \$targetDir
  Start-Process -FilePath \$targetExe
  throw
} finally {
  if (Test-Path \$work) { Remove-Item \$work -Recurse -Force }
}
''';
}

String _shellQuote(String value) {
  return "'${value.replaceAll("'", "'\"'\"'")}'";
}

String _powerShellQuote(String value) {
  return "'${value.replaceAll("'", "''")}'";
}
