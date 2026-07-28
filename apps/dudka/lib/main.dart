import 'dart:io';

import 'package:flutter/material.dart';

import 'app.dart';
import 'desktop/autostart_service.dart';
import 'desktop/desktop_lifecycle.dart';
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
Future<void> main(List<String> arguments) async {
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

  DesktopLifecycle? desktop;
  if (Platform.isMacOS || Platform.isWindows || Platform.isLinux) {
    desktop = DesktopLifecycle(
      autostart: DesktopAutostartService(),
      beforeExit: () async {
        await hostedEngine?.stop();
      },
    );
    await desktop.initialize(startHidden: arguments.contains('--hidden'));
  }

  runApp(DudkaApp(
    engineBase: engineBase,
    client: EngineClient(baseUrl: engineBase),
    updater: updater,
    desktop: desktop,
  ));
}

Directory _defaultEngineDataDir() {
  final home = Platform.environment[Platform.isWindows ? 'APPDATA' : 'HOME'];
  if (home != null && home.isNotEmpty && Platform.isMacOS) {
    return Directory('$home/Library/Application Support/dudka/flutter-engine');
  }
  if (home != null && home.isNotEmpty && Platform.isWindows) {
    return Directory('$home\\Dudka\\engine');
  }
  if (home != null && home.isNotEmpty) {
    return Directory('$home/.local/share/dudka/engine');
  }
  return Directory('${Directory.systemTemp.path}/dudka-flutter-engine');
}
