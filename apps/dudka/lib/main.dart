import 'dart:io';

import 'package:flutter/material.dart';

import 'app.dart';
import 'engine/bundle.dart';
import 'engine/client.dart';
import 'engine/host.dart';
import 'update/update_manager.dart';

/// macOS-first shell (P061/P081).
///
/// Engine URL resolution order:
/// 1. `--dart-define=DUDKA_ENGINE=http://127.0.0.1:PORT` (external dudkad)
/// 2. `--dart-define=DUDKAD_BIN=/path/to/dudkad`
/// 3. bundled `dudkad` next to the app executable (`.app/Contents/MacOS/dudkad`)
/// 4. default attach `http://127.0.0.1:17880`
Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  const predefined = String.fromEnvironment('DUDKA_ENGINE');
  const binDefine = String.fromEnvironment('DUDKAD_BIN');

  late final String engineBase;
  EngineHost? hostedEngine;

  if (predefined.isNotEmpty) {
    engineBase = predefined;
  } else {
    final bin = binDefine.isNotEmpty ? binDefine : resolveBundledDudkadBin();
    if (bin != null && bin.isNotEmpty) {
      final dataDir = _defaultEngineDataDir();
      hostedEngine =
          EngineHost(binaryPath: bin, dataDir: dataDir.path, name: 'ДУДКА');
      engineBase = await hostedEngine.start();
    } else {
      engineBase = 'http://127.0.0.1:17880';
    }
  }

  UpdateManager? updater;
  try {
    updater = await UpdateManager.forCurrentPlatform(
      beforeExit: () async {
        await hostedEngine?.stop();
      },
    );
  } catch (_) {
    // Update availability must never block the LAN chat.
  }

  runApp(DudkaApp(
    engineBase: engineBase,
    client: EngineClient(baseUrl: engineBase),
    updater: updater,
  ));
}

Directory _defaultEngineDataDir() {
  final home = Platform.environment['HOME'];
  if (home != null && home.isNotEmpty) {
    return Directory('$home/Library/Application Support/dudka/flutter-engine');
  }
  return Directory('${Directory.systemTemp.path}/dudka-flutter-engine');
}
