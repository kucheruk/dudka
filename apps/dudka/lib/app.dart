import 'dart:io';

import 'package:flutter/material.dart';

import 'engine/client.dart';
import 'desktop/desktop_lifecycle.dart';
import 'nick/fallback.dart';
import 'screens/chat_screen.dart';
import 'screens/first_run_nick_screen.dart';
import 'screens/internet_consent_screen.dart';
import 'session/first_run_store.dart';
import 'storage/app_paths.dart';
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
  bool? _internetConfirmed;
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
    var internetConfirmed = false;
    try {
      final me = await _client.fetchMe();
      suggested = me.name;
      internetConfirmed = await _client.fetchInternetConsent();
    } catch (_) {}
    if (!mounted) return;
    final confirmed = _store.isNickConfirmed();
    final legacyPlaceholder =
        confirmed && suggested.trim().toUpperCase() == 'ДУДКА';
    // 0.4.0 and older forced the product name into the user's nick on every
    // GUI start. Ask once again when that legacy placeholder is detected.
    setState(() {
      _suggested = legacyPlaceholder ? '' : suggested;
      _nickConfirmed = legacyPlaceholder ? false : confirmed;
      _internetConfirmed = internetConfirmed;
    });
  }

  Directory _defaultShellDir() => DudkaAppPaths.shellDataDir();

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
    if (confirmed == null || _internetConfirmed == null) {
      return const Scaffold(
        body: Center(
          child: CircularProgressIndicator(key: Key('boot-loading')),
        ),
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
        onDone: (nick) async {
          await _store.markNickConfirmed(nick);
          if (!mounted) return;
          setState(() => _nickConfirmed = true);
        },
      );
    }
    if (!_internetConfirmed!) {
      return InternetConsentScreen(
        client: _client,
        onDone: () async {
          await _store.markInternetConfirmed();
          if (!mounted) return;
          setState(() => _internetConfirmed = true);
        },
      );
    }
    return ChatScreen(
      client: _client,
      pollInterval: widget.chatPollInterval,
      updater: widget.updater,
      desktop: widget.desktop,
      onNickChanged: (nick) => _store.markNickConfirmed(nick),
    );
  }
}
