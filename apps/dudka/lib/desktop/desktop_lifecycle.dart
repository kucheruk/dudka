import 'dart:async';
import 'dart:io';

import 'package:flutter/services.dart';
import 'package:tray_manager/tray_manager.dart';
import 'package:window_manager/window_manager.dart';

import '../engine/client.dart';
import 'autostart_service.dart';
import 'unread_tracker.dart';

abstract interface class DesktopLifecycleHandle {
  AutostartController get autostart;
  void observeSnapshot(ChatSnapshot snapshot);
}

class DesktopLifecycle
    with WindowListener, TrayListener
    implements DesktopLifecycleHandle {
  DesktopLifecycle({
    required this.autostart,
    required this.beforeExit,
    MethodChannel badgeChannel =
        const MethodChannel('team.zamoo.dudka/desktop'),
  }) : _badgeChannel = badgeChannel;

  @override
  final AutostartController autostart;
  final Future<void> Function() beforeExit;
  final MethodChannel _badgeChannel;
  final UnreadTracker _unread = UnreadTracker();

  bool _active = true;
  bool _quitting = false;
  int _lastBadge = -1;

  Future<void> initialize({required bool startHidden}) async {
    await windowManager.ensureInitialized();
    windowManager.addListener(this);
    trayManager.addListener(this);
    await windowManager.setPreventClose(true);
    await trayManager.setIcon(
      Platform.isWindows
          ? 'windows/runner/resources/app_icon.ico'
          : 'assets/branding/app_icon_master.png',
    );
    await trayManager.setToolTip('ДУДКА');
    await trayManager.setContextMenu(
      Menu(
        items: [
          MenuItem(
            key: 'open',
            label: 'Открыть Дудку',
            onClick: (_) => unawaited(showWindow()),
          ),
          MenuItem.separator(),
          MenuItem(
            key: 'exit',
            label: 'Выйти',
            onClick: (_) => unawaited(quit()),
          ),
        ],
      ),
    );

    const options = WindowOptions(
      title: 'ДУДКА',
      minimumSize: Size(640, 480),
      size: Size(1120, 720),
      center: true,
    );
    await windowManager.waitUntilReadyToShow(options, () async {
      if (startHidden) {
        _active = false;
        await windowManager.hide();
      } else {
        await showWindow();
      }
    });
  }

  @override
  void observeSnapshot(ChatSnapshot snapshot) {
    final count = _unread.observe(
      messages: snapshot.messages,
      selfPeerId: snapshot.me.peerId,
      active: _active,
    );
    unawaited(_showBadge(count));
  }

  Future<void> showWindow() async {
    await windowManager.show();
    await windowManager.focus();
    _active = true;
    await _showBadge(_unread.clear());
  }

  Future<void> quit() async {
    if (_quitting) return;
    _quitting = true;
    windowManager.removeListener(this);
    trayManager.removeListener(this);
    await _showBadge(0);
    await trayManager.destroy();
    await beforeExit();
    await windowManager.setPreventClose(false);
    await windowManager.destroy();
    exit(0);
  }

  Future<void> _showBadge(int count) async {
    if (_lastBadge == count) return;
    _lastBadge = count;
    await trayManager.setToolTip(
      count == 0 ? 'ДУДКА' : 'ДУДКА · непрочитано $count',
    );
    try {
      await _badgeChannel.invokeMethod<void>('setBadge', count);
    } on MissingPluginException {
      // The tray tooltip remains a truthful fallback on unsupported desktops.
    } on PlatformException {
      // A desktop shell may not implement launcher badges.
    }
  }

  @override
  void onWindowClose() {
    if (_quitting) return;
    _active = false;
    unawaited(windowManager.hide());
  }

  @override
  void onWindowFocus() {
    _active = true;
    unawaited(_showBadge(_unread.clear()));
  }

  @override
  void onWindowBlur() {
    _active = false;
  }

  @override
  void onTrayIconMouseDown() {
    unawaited(showWindow());
  }

  @override
  void onTrayMenuItemClick(MenuItem menuItem) {
    if (menuItem.key == 'open') unawaited(showWindow());
    if (menuItem.key == 'exit') unawaited(quit());
  }
}
