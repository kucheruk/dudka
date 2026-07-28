import 'dart:io';

enum DesktopSystem { macos, windows, linux }

typedef ProcessRunner = Future<ProcessResult> Function(
  String executable,
  List<String> arguments,
);

abstract interface class AutostartController {
  Future<bool> isEnabled();
  Future<void> setEnabled(bool enabled);
}

class DesktopAutostartService implements AutostartController {
  DesktopAutostartService({
    DesktopSystem? system,
    String? executablePath,
    String? homePath,
    ProcessRunner? runProcess,
  })  : system = system ?? currentDesktopSystem(),
        executablePath = executablePath ?? Platform.resolvedExecutable,
        homePath = homePath ?? _homePath(),
        _runProcess = runProcess ?? Process.run;

  static const _windowsRunKey =
      r'HKCU\Software\Microsoft\Windows\CurrentVersion\Run';
  static const _windowsValue = 'Dudka';

  final DesktopSystem system;
  final String executablePath;
  final String homePath;
  final ProcessRunner _runProcess;

  File get _macPlist =>
      File('$homePath/Library/LaunchAgents/team.zamoo.dudka.plist');

  File get _linuxDesktop =>
      File('$homePath/.config/autostart/team.zamoo.dudka.desktop');

  @override
  Future<bool> isEnabled() async {
    switch (system) {
      case DesktopSystem.macos:
        return _macPlist.exists();
      case DesktopSystem.windows:
        final result = await _runProcess(
          'reg.exe',
          ['query', _windowsRunKey, '/v', _windowsValue],
        );
        return result.exitCode == 0;
      case DesktopSystem.linux:
        return _linuxDesktop.exists();
    }
  }

  @override
  Future<void> setEnabled(bool enabled) async {
    switch (system) {
      case DesktopSystem.macos:
        await _setMac(enabled);
        return;
      case DesktopSystem.windows:
        await _setWindows(enabled);
        return;
      case DesktopSystem.linux:
        await _setLinux(enabled);
        return;
    }
  }

  Future<void> _setMac(bool enabled) async {
    final file = _macPlist;
    if (!enabled) {
      if (await file.exists()) await file.delete();
      return;
    }
    await file.parent.create(recursive: true);
    await file.writeAsString('''<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>team.zamoo.dudka</string>
  <key>ProgramArguments</key>
  <array>
    <string>${_xmlEscape(executablePath)}</string>
    <string>--hidden</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
</dict>
</plist>
''');
  }

  Future<void> _setWindows(bool enabled) async {
    final arguments = enabled
        ? [
            'add',
            _windowsRunKey,
            '/v',
            _windowsValue,
            '/t',
            'REG_SZ',
            '/d',
            '"$executablePath" --hidden',
            '/f',
          ]
        : ['delete', _windowsRunKey, '/v', _windowsValue, '/f'];
    final result = await _runProcess('reg.exe', arguments);
    if (result.exitCode != 0 && (enabled || result.exitCode != 1)) {
      throw ProcessException(
        'reg.exe',
        arguments,
        '${result.stderr}',
        result.exitCode,
      );
    }
  }

  Future<void> _setLinux(bool enabled) async {
    final file = _linuxDesktop;
    if (!enabled) {
      if (await file.exists()) await file.delete();
      return;
    }
    await file.parent.create(recursive: true);
    await file.writeAsString('''[Desktop Entry]
Type=Application
Name=ДУДКА
Comment=Локальный квартирный чат
Exec=${_desktopExecQuote(executablePath)} --hidden
Icon=team.zamoo.dudka
Terminal=false
X-GNOME-Autostart-enabled=true
''');
  }
}

DesktopSystem currentDesktopSystem() {
  if (Platform.isMacOS) return DesktopSystem.macos;
  if (Platform.isWindows) return DesktopSystem.windows;
  if (Platform.isLinux) return DesktopSystem.linux;
  throw UnsupportedError('autostart is desktop-only');
}

String _homePath() {
  final value =
      Platform.environment[Platform.isWindows ? 'USERPROFILE' : 'HOME'];
  if (value == null || value.trim().isEmpty) {
    throw StateError('Не найдена домашняя папка пользователя');
  }
  return value;
}

String _xmlEscape(String value) => value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&apos;');

String _desktopExecQuote(String value) =>
    '"${value.replaceAll(r'\', r'\\').replaceAll('"', r'\"')}"';
