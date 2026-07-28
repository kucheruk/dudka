import 'dart:io';

/// Stable user-owned data locations. They live outside the application bundle,
/// so replacing or reinstalling the app does not remove identity or history.
class DudkaAppPaths {
  const DudkaAppPaths._();

  static Directory engineDataDir() {
    final home = Platform.environment[Platform.isWindows ? 'APPDATA' : 'HOME'];
    if (home != null && home.isNotEmpty && Platform.isMacOS) {
      return Directory(
        '$home/Library/Application Support/dudka/flutter-engine',
      );
    }
    if (home != null && home.isNotEmpty && Platform.isWindows) {
      return Directory('$home\\Dudka\\engine');
    }
    if (home != null && home.isNotEmpty) {
      return Directory('$home/.local/share/dudka/engine');
    }
    return Directory('${Directory.systemTemp.path}/dudka-flutter-engine');
  }

  static Directory shellDataDir() {
    final home = Platform.environment[Platform.isWindows ? 'APPDATA' : 'HOME'];
    if (home != null && home.isNotEmpty && Platform.isMacOS) {
      return Directory('$home/Library/Application Support/dudka/flutter-shell');
    }
    if (home != null && home.isNotEmpty && Platform.isWindows) {
      return Directory('$home\\Dudka\\shell');
    }
    if (home != null && home.isNotEmpty) {
      return Directory('$home/.local/share/dudka/shell');
    }
    return Directory('${Directory.systemTemp.path}/dudka-flutter-shell');
  }
}
