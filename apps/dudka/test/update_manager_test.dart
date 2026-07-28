import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:dudka/update/update_manager.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

class FakeUpdateActivator implements UpdateActivator {
  int calls = 0;
  UpdatePackage? package;
  File? archive;
  String? version;
  Duration? delay;

  @override
  Future<void> schedule({
    required UpdatePackage package,
    required File archive,
    required String version,
    required Duration delay,
  }) async {
    calls++;
    this.package = package;
    this.archive = archive;
    this.version = version;
    this.delay = delay;
  }
}

Map<String, Object?> manifestJson({
  required List<int> artifact,
  String version = '0.3.1',
  String platform = 'macos-universal',
  String? hash,
  int? size,
  String url =
      'https://zamoo.team/dudka/releases/dudka-macos-universal.zip',
}) {
  return {
    'schema': 1,
    'version': version,
    'packages': {
      platform: {
        'url': url,
        'sha256': hash ?? sha256.convert(artifact).toString(),
        'size': size ?? artifact.length,
        'format': 'zip',
      },
    },
  };
}

void main() {
  group('manifest contract', () {
    test('parses a strict HTTPS package', () {
      final manifest = UpdateManifest.fromJson(
        manifestJson(artifact: utf8.encode('archive')),
      );

      expect(manifest.version, '0.3.1');
      expect(
        manifest.packages['macos-universal']?.url,
        Uri.parse(
          'https://zamoo.team/dudka/releases/dudka-macos-universal.zip',
        ),
      );
    });

    test('rejects unsupported schema and unsafe package fields', () {
      expect(
        () => UpdateManifest.fromJson({
          ...manifestJson(artifact: utf8.encode('archive')),
          'schema': 2,
        }),
        throwsFormatException,
      );
      expect(
        () => UpdateManifest.fromJson(
          manifestJson(
            artifact: utf8.encode('archive'),
            url: 'http://zamoo.team/dudka.zip',
          ),
        ),
        throwsFormatException,
      );
      expect(
        () => UpdateManifest.fromJson(
          manifestJson(
            artifact: utf8.encode('archive'),
            hash: 'not-a-sha256',
          ),
        ),
        throwsFormatException,
      );
      expect(
        () => UpdateManifest.fromJson(
          manifestJson(
            artifact: utf8.encode('archive'),
            size: 0,
          ),
        ),
        throwsFormatException,
      );
    });
  });

  test('compares stable SemVer without prerelease ambiguity', () {
    expect(isVersionNewer('0.3.0', '0.2.0'), isTrue);
    expect(isVersionNewer('1.0.0', '0.99.99'), isTrue);
    expect(isVersionNewer('0.3.0', '0.3.0'), isFalse);
    expect(isVersionNewer('0.2.9', '0.3.0'), isFalse);
    expect(isVersionNewer('0.3.1-beta', '0.3.0'), isFalse);
  });

  test('downloads, verifies, exposes ready, and schedules exact delay',
      () async {
    final artifact = utf8.encode('verified zip fixture');
    final temp = await Directory.systemTemp.createTemp('dudka-update-test-');
    final activator = FakeUpdateActivator();
    var beforeExitCalls = 0;
    int? exitCode;
    final client = MockClient((request) async {
      if (request.url.path == '/dudka/update.json') {
        return http.Response(
          jsonEncode(manifestJson(artifact: artifact)),
          HttpStatus.ok,
        );
      }
      if (request.url.path ==
          '/dudka/releases/dudka-macos-universal.zip') {
        return http.Response.bytes(artifact, HttpStatus.ok);
      }
      return http.Response('missing', HttpStatus.notFound);
    });
    final manager = UpdateManager(
      currentVersion: '0.3.0',
      platformKey: 'macos-universal',
      cacheRoot: temp,
      activator: activator,
      beforeExit: () async => beforeExitCalls++,
      exitApplication: (code) => exitCode = code,
      client: client,
      checkInterval: Duration.zero,
    );
    addTearDown(() async {
      manager.dispose();
      if (await temp.exists()) await temp.delete(recursive: true);
    });

    await manager.checkAndStage();

    expect(manager.snapshot.phase, UpdatePhase.ready);
    expect(manager.snapshot.version, '0.3.1');
    final archive = File(
      '${temp.path}${Platform.pathSeparator}0.3.1'
      '${Platform.pathSeparator}macos-universal.zip',
    );
    expect(await archive.readAsBytes(), artifact);

    await manager.activate();

    expect(activator.calls, 1);
    expect(activator.archive?.path, archive.path);
    expect(activator.version, '0.3.1');
    expect(activator.delay, const Duration(seconds: 10));
    expect(beforeExitCalls, 1);
    expect(exitCode, 0);
  });

  test('bad artifact never becomes ready and removes partial download',
      () async {
    final artifact = utf8.encode('tampered zip fixture');
    final temp = await Directory.systemTemp.createTemp('dudka-update-test-');
    final client = MockClient((request) async {
      if (request.url.path == '/dudka/update.json') {
        return http.Response(
          jsonEncode(
            manifestJson(
              artifact: artifact,
              hash: '0' * 64,
            ),
          ),
          HttpStatus.ok,
        );
      }
      return http.Response.bytes(artifact, HttpStatus.ok);
    });
    final manager = UpdateManager(
      currentVersion: '0.3.0',
      platformKey: 'macos-universal',
      cacheRoot: temp,
      activator: FakeUpdateActivator(),
      beforeExit: () async {},
      exitApplication: (_) {},
      client: client,
      checkInterval: Duration.zero,
    );
    addTearDown(() async {
      manager.dispose();
      if (await temp.exists()) await temp.delete(recursive: true);
    });

    await manager.checkAndStage();

    expect(manager.snapshot.phase, UpdatePhase.failed);
    expect(manager.snapshot.isReady, isFalse);
    expect(
      File(
        '${temp.path}${Platform.pathSeparator}0.3.1'
        '${Platform.pathSeparator}macos-universal.zip.partial',
      ).existsSync(),
      isFalse,
    );
  });

  test('current version and absent platform stay unavailable', () async {
    final artifact = utf8.encode('archive');
    final temp = await Directory.systemTemp.createTemp('dudka-update-test-');
    final client = MockClient((_) async {
      return http.Response(
        jsonEncode(
          manifestJson(
            artifact: artifact,
            version: '0.3.0',
            platform: 'windows-amd64',
          ),
        ),
        HttpStatus.ok,
      );
    });
    final manager = UpdateManager(
      currentVersion: '0.3.0',
      platformKey: 'macos-universal',
      cacheRoot: temp,
      activator: FakeUpdateActivator(),
      beforeExit: () async {},
      exitApplication: (_) {},
      client: client,
      checkInterval: Duration.zero,
    );
    addTearDown(() async {
      manager.dispose();
      if (await temp.exists()) await temp.delete(recursive: true);
    });

    await manager.checkAndStage();

    expect(manager.snapshot.phase, UpdatePhase.unavailable);
  });

  test('platform helpers wait, replace with backup, and relaunch', () {
    final mac = buildMacActivationScript(
      archivePath: "/tmp/Dudka's update.zip",
      targetBundlePath: '/Applications/dudka.app',
      processId: 42,
      delay: const Duration(seconds: 10),
      version: '0.3.1',
    );
    expect(mac, contains('sleep 10'));
    expect(mac, contains('while kill -0 42'));
    expect(mac, contains('.backup-0.3.1'));
    expect(mac, contains('/usr/bin/open'));
    expect(mac, contains("'\"'\"'"));

    final windows = buildWindowsActivationScript(
      archivePath: r"C:\Temp\Dudka's update.zip",
      targetExecutablePath: r'C:\Apps\dudka\dudka.exe',
      processId: 42,
      delay: const Duration(seconds: 10),
      version: '0.3.1',
    );
    expect(windows, contains('Start-Sleep -Seconds 10'));
    expect(windows, contains('Wait-Process -Id 42'));
    expect(windows, contains('dudka-backup-0.3.1'));
    expect(windows, contains('Start-Process -FilePath \$targetExe'));
    expect(windows, contains("Dudka''s update.zip"));
  });

  test('generated mac helper is valid POSIX shell', () async {
    if (Platform.isWindows) return;
    final temp = await Directory.systemTemp.createTemp('dudka-helper-test-');
    addTearDown(() async {
      if (await temp.exists()) await temp.delete(recursive: true);
    });
    final script = File('${temp.path}/activate.sh');
    await script.writeAsString(buildMacActivationScript(
      archivePath: "/tmp/Dudka's update.zip",
      targetBundlePath: '/Applications/dudka.app',
      processId: 42,
      delay: const Duration(seconds: 10),
      version: '0.3.1',
    ));

    final result = await Process.run('/bin/sh', ['-n', script.path]);

    expect(result.exitCode, 0, reason: result.stderr.toString());
  });

  test('mac bundle is derived from the resolved executable', () {
    expect(
      macBundleFromExecutable(
        '/Applications/dudka.app/Contents/MacOS/dudka',
      ).path,
      '/Applications/dudka.app',
    );
    expect(
      () => macBundleFromExecutable('/tmp/dudka'),
      throwsStateError,
    );
  });
}
