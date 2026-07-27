import 'dart:io';

import 'package:flutter/material.dart';

import 'app.dart';
import 'engine/client.dart';
import 'engine/host.dart';

/// macOS-first skeleton (P061).
///
/// Engine URL resolution order:
/// 1. `--dart-define=DUDKA_ENGINE=http://127.0.0.1:PORT` (external dudkad)
/// 2. spawn `dudkad` via `--dart-define=DUDKAD_BIN=/path/to/dudkad`
/// 3. default attach `http://127.0.0.1:17880`
Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  const predefined = String.fromEnvironment('DUDKA_ENGINE');
  const binDefine = String.fromEnvironment('DUDKAD_BIN');

  late final String engineBase;

  if (predefined.isNotEmpty) {
    engineBase = predefined;
  } else if (binDefine.isNotEmpty) {
    final dataDir = _defaultEngineDataDir();
    final host = EngineHost(binaryPath: binDefine, dataDir: dataDir.path, name: 'ДУДКА');
    engineBase = await host.start();
  } else {
    engineBase = 'http://127.0.0.1:17880';
  }

  runApp(DudkaApp(engineBase: engineBase, client: EngineClient(baseUrl: engineBase)));
}

Directory _defaultEngineDataDir() {
  final home = Platform.environment['HOME'];
  if (home != null && home.isNotEmpty) {
    return Directory('$home/Library/Application Support/dudka/flutter-engine');
  }
  return Directory('${Directory.systemTemp.path}/dudka-flutter-engine');
}
