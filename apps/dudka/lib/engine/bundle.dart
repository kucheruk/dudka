import 'dart:io';

/// Locate packaged `dudkad` inside a desktop bundle (P081/P156).
String? resolveBundledDudkadBin({String? executablePath}) {
  try {
    final exeDir = File(executablePath ?? Platform.resolvedExecutable).parent;
    final windowsInternal = File(
      '${exeDir.path}${Platform.pathSeparator}internal'
      '${Platform.pathSeparator}dudkad.exe',
    );
    if (windowsInternal.existsSync()) return windowsInternal.path;
    final sibling = File('${exeDir.path}/dudkad');
    if (sibling.existsSync()) return sibling.path;
    final resources = Directory('${exeDir.parent.path}/Resources');
    final resBin = File('${resources.path}/dudkad');
    if (resBin.existsSync()) return resBin.path;
  } catch (_) {}
  return null;
}
