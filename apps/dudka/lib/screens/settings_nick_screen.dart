import 'package:flutter/material.dart';

import '../engine/client.dart';
import '../desktop/autostart_service.dart';
import '../theme/dudka_theme.dart';

/// Mini-settings: только ник (P066 / DUD-UI-115). Никаких лишних полей профиля.
class SettingsNickScreen extends StatefulWidget {
  const SettingsNickScreen({
    super.key,
    required this.client,
    required this.initialNick,
    this.autostart,
  });

  final EngineClient client;
  final String initialNick;
  final AutostartController? autostart;

  @override
  State<SettingsNickScreen> createState() => _SettingsNickScreenState();
}

class _SettingsNickScreenState extends State<SettingsNickScreen> {
  late final TextEditingController _ctrl;
  String? _error;
  bool _busy = false;
  bool _autostartBusy = false;
  bool _autostartEnabled = false;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: widget.initialNick);
    _loadAutostart();
  }

  Future<void> _loadAutostart() async {
    final autostart = widget.autostart;
    if (autostart == null) return;
    try {
      final enabled = await autostart.isEnabled();
      if (!mounted) return;
      setState(() => _autostartEnabled = enabled);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = 'Не удалось прочитать автозапуск: $error');
    }
  }

  Future<void> _setAutostart(bool enabled) async {
    final autostart = widget.autostart;
    if (autostart == null || _autostartBusy) return;
    setState(() {
      _autostartBusy = true;
      _error = null;
    });
    try {
      await autostart.setEnabled(enabled);
      if (!mounted) return;
      setState(() => _autostartEnabled = enabled);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = 'Не удалось изменить автозапуск: $error');
    } finally {
      if (mounted) setState(() => _autostartBusy = false);
    }
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (_busy) return;
    final nick = _ctrl.text.trim();
    if (nick.isEmpty) {
      setState(() => _error = 'Укажите ник');
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.client.setNick(nick);
      if (!mounted) return;
      Navigator.of(context).pop(nick);
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Настройки'),
        leading: IconButton(
          key: const Key('settings-back'),
          icon: const Icon(Icons.arrow_back),
          onPressed: _busy ? null : () => Navigator.of(context).pop(),
        ),
      ),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text('НИК', style: DudkaType.label()),
            const SizedBox(height: 8),
            Text(
              'Так вас видят соседи в чате. Больше ничего настраивать не нужно.',
              style: DudkaType.mono(size: 13, color: DudkaColors.silkscreenDim),
            ),
            const SizedBox(height: 24),
            TextField(
              key: const Key('settings-nick-field'),
              controller: _ctrl,
              enabled: !_busy,
              style: DudkaType.mono(),
              decoration: const InputDecoration(labelText: 'Ник'),
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _save(),
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!,
                  style: DudkaType.mono(size: 12, color: DudkaColors.danger)),
            ],
            const SizedBox(height: 24),
            FilledButton(
              key: const Key('settings-nick-save'),
              onPressed: _busy ? null : _save,
              child: const Text('Сохранить'),
            ),
            if (widget.autostart != null) ...[
              const SizedBox(height: 32),
              Text('ПРИЛОЖЕНИЕ', style: DudkaType.label()),
              const SizedBox(height: 8),
              SwitchListTile(
                key: const Key('settings-autostart'),
                contentPadding: EdgeInsets.zero,
                value: _autostartEnabled,
                onChanged: _autostartBusy ? null : _setAutostart,
                title: const Text('Запускать при входе в систему'),
                subtitle: const Text('Дудка запустится скрытой в трее.'),
              ),
              const SizedBox(height: 8),
              Text(
                'Крестик прячет окно в трей. Для полного выхода выберите '
                '«Выйти» в меню Дудки.',
                style: DudkaType.mono(
                  size: 12,
                  color: DudkaColors.silkscreenDim,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
