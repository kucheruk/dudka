import 'dart:io';

import 'package:dudka/engine/bundle.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('resolveBundledDudkadBin is callable (P081)', () {
    // In unit tests there is no .app bundle — function must not throw.
    expect(() => resolveBundledDudkadBin(), returnsNormally);
  });

  test('portable Windows bundle keeps engine in internal directory', () async {
    final root = await Directory.systemTemp.createTemp('dudka-portable-');
    addTearDown(() => root.delete(recursive: true));
    final gui = File('${root.path}${Platform.pathSeparator}dudka.exe');
    final engine = File(
      '${root.path}${Platform.pathSeparator}internal'
      '${Platform.pathSeparator}dudkad.exe',
    );
    await engine.parent.create(recursive: true);
    await gui.writeAsBytes(const [0]);
    await engine.writeAsBytes(const [0]);

    expect(resolveBundledDudkadBin(executablePath: gui.path), engine.path);
  });
}
