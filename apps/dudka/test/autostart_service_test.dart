import 'dart:io';

import 'package:dudka/desktop/autostart_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('macOS writes and removes a user LaunchAgent', () async {
    final home = await Directory.systemTemp.createTemp('dudka-autostart-mac-');
    addTearDown(() => home.delete(recursive: true));
    final service = DesktopAutostartService(
      system: DesktopSystem.macos,
      executablePath: '/Applications/ДУДКА.app/Contents/MacOS/dudka',
      homePath: home.path,
    );

    expect(await service.isEnabled(), isFalse);
    await service.setEnabled(true);
    expect(await service.isEnabled(), isTrue);
    final contents = await File(
      '${home.path}/Library/LaunchAgents/team.zamoo.dudka.plist',
    ).readAsString();
    expect(contents, contains('<string>--hidden</string>'));
    expect(contents, contains('/Applications/ДУДКА.app/Contents/MacOS/dudka'));

    await service.setEnabled(false);
    expect(await service.isEnabled(), isFalse);
  });

  test('Linux writes a non-terminal hidden-start desktop entry', () async {
    final home =
        await Directory.systemTemp.createTemp('dudka-autostart-linux-');
    addTearDown(() => home.delete(recursive: true));
    final service = DesktopAutostartService(
      system: DesktopSystem.linux,
      executablePath: '/opt/Дудка/dudka',
      homePath: home.path,
    );

    await service.setEnabled(true);
    final contents = await File(
      '${home.path}/.config/autostart/team.zamoo.dudka.desktop',
    ).readAsString();
    expect(contents, contains('Exec="/opt/Дудка/dudka" --hidden'));
    expect(contents, contains('Terminal=false'));
  });

  test('Windows uses the current-user Run key with one GUI command', () async {
    final calls = <List<String>>[];
    var enabled = false;
    Future<ProcessResult> runner(
        String executable, List<String> arguments) async {
      calls.add([executable, ...arguments]);
      if (arguments.first == 'query') {
        return ProcessResult(1, enabled ? 0 : 1, '', '');
      }
      enabled = arguments.first == 'add';
      return ProcessResult(1, 0, '', '');
    }

    final service = DesktopAutostartService(
      system: DesktopSystem.windows,
      executablePath: r'C:\Users\Me\AppData\Local\Programs\Dudka\dudka.exe',
      homePath: r'C:\Users\Me',
      runProcess: runner,
    );

    expect(await service.isEnabled(), isFalse);
    await service.setEnabled(true);
    expect(await service.isEnabled(), isTrue);
    expect(
      calls.expand((call) => call),
      contains(
        r'"C:\Users\Me\AppData\Local\Programs\Dudka\dudka.exe" --hidden',
      ),
    );
  });
}
