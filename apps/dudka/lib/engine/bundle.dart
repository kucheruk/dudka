import 'dart:io';

/// Locate packaged `dudkad` beside the Flutter executable (P081).
String? resolveBundledDudkadBin() {
  try {
    final exeDir = File(Platform.resolvedExecutable).parent;
    final sibling = File('${exeDir.path}/dudkad');
    if (sibling.existsSync()) return sibling.path;
    final resources = Directory('${exeDir.parent.path}/Resources');
    final resBin = File('${resources.path}/dudkad');
    if (resBin.existsSync()) return resBin.path;
  } catch (_) {}
  return null;
}
