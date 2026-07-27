import 'package:dudka/engine/bundle.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('resolveBundledDudkadBin is callable (P081)', () {
    // In unit tests there is no .app bundle — function must not throw.
    expect(() => resolveBundledDudkadBin(), returnsNormally);
  });
}
