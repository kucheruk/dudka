import 'dart:io';

import 'package:flutter/material.dart';

import 'engine/client.dart';
import 'desktop/desktop_lifecycle.dart';
import 'nick/fallback.dart';
import 'screens/chat_screen.dart';
import 'screens/first_run_nick_screen.dart';
import 'session/first_run_store.dart';
import 'theme/dudka_theme.dart';
import 'update/update_manager.dart';

/// Flutter shell (P061/P062): first-run nick → chat.
class DudkaApp extends StatefulWidget {
  const DudkaApp({
    super.key,
    required this.engineBase,
    this.client,
    this.firstRunStore,
    this.hostnameForFallback,
    this.nickPick,
    this.chatPollInterval = const Duration(seconds: 1),
    this.updater,
    this.desktop,
  });

  final String engineBase;
  final EngineClient? client;
  final FirstRunStore? firstRunStore;
  final String Function()? hostnameForFallback;
  final NickPick? nickPick;
  final Duration chatPollInterval;
  final UpdateController? updater;
  final DesktopLifecycleHandle? desktop;

  @override
  State<DudkaApp> createState() => _DudkaAppState();
}

class _DudkaAppState extends State<DudkaApp> {
  late final EngineClient _client;
  late final FirstRunStore _store;
  bool? _nickConfirmed;
  String _suggested = '';

  @override
  void initState() {
    super.initState();
    _client = widget.client ?? EngineClient(baseUrl: widget.engineBase);
    _store = widget.firstRunStore ?? FirstRunStore.inDir(_defaultShellDir());
    _bootstrap();
  }

  Future<void> _bootstrap() async {
    var suggested = '';
    try {
      final me = await _client.fetchMe();
      suggested = me.name;
    } catch (_) {}
    if (!mounted) return;
    // Re-read store after await — first-run may have completed meanwhile.
    setState(() {
      _suggested = suggested;
      _nickConfirmed = _store.isNickConfirmed();
    });
  }

  Directory _defaultShellDir() {
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

  @override
  void dispose() {
    if (widget.client == null) {
      _client.close();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'ДУДКА',
      debugShowCheckedModeBanner: false,
      theme: buildDudkaTheme(),
      home: _home(),
    );
  }

  Widget _home() {
    final confirmed = _nickConfirmed;
    if (confirmed == null) {
      return const Scaffold(
        body:
            Center(child: CircularProgressIndicator(key: Key('boot-loading'))),
      );
    }
    if (!confirmed) {
      return FirstRunNickScreen(
        client: _client,
        suggested: _suggested,
        hostnameForFallback: widget.hostnameForFallback ??
            () {
              try {
                return Platform.localHostname;
              } catch (_) {
                return '';
              }
            },
        nickPick: widget.nickPick,
        onDone: (_) async {
          await _store.markNickConfirmed();
          if (!mounted) return;
          setState(() => _nickConfirmed = true);
        },
      );
    }
    return ChatScreen(
      client: _client,
      pollInterval: widget.chatPollInterval,
      updater: widget.updater,
      desktop: widget.desktop,
    );
  }
}
