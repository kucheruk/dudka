import 'dart:io';

import 'package:flutter/material.dart';

import 'app.dart';
import 'desktop/autostart_service.dart';
import 'desktop/desktop_lifecycle.dart';
import 'engine/bundle.dart';
import 'engine/client.dart';
import 'engine/host.dart';
import 'session/first_run_store.dart';
import 'storage/app_paths.dart';
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
  final firstRunStore = FirstRunStore.inDir(DudkaAppPaths.shellDataDir());

  if (predefined.isNotEmpty) {
    engineBase = predefined;
  } else {
    final bin = binDefine.isNotEmpty ? binDefine : resolveBundledDudkadBin();
    if (bin != null && bin.isNotEmpty) {
      final dataDir = DudkaAppPaths.engineDataDir();
      final engineNameFile =
          File('${dataDir.path}${Platform.pathSeparator}display_name');
      var engineHasName = false;
      try {
        engineHasName = engineNameFile.existsSync() &&
            engineNameFile.readAsStringSync().trim().isNotEmpty;
      } catch (_) {
        // Let the engine report an unreadable identity file itself.
      }
      hostedEngine = EngineHost(
        binaryPath: bin,
        dataDir: dataDir.path,
        name: engineHasName ? '' : firstRunStore.confirmedNick() ?? '',
      );
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

  runApp(
    DudkaApp(
      engineBase: engineBase,
      client: EngineClient(baseUrl: engineBase),
      firstRunStore: firstRunStore,
      updater: updater,
      desktop: desktop,
    ),
  );
}
