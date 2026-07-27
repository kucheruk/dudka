import 'package:flutter/material.dart';

import '../engine/client.dart';

/// Mini-settings: только ник (P066 / DUD-UI-115). Никаких лишних полей профиля.
class SettingsNickScreen extends StatefulWidget {
  const SettingsNickScreen({
    super.key,
    required this.client,
    required this.initialNick,
  });

  final EngineClient client;
  final String initialNick;

  @override
  State<SettingsNickScreen> createState() => _SettingsNickScreenState();
}

class _SettingsNickScreenState extends State<SettingsNickScreen> {
  late final TextEditingController _ctrl;
  String? _error;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: widget.initialNick);
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
            const Text(
              'Ник',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            const Text(
              'Так вас видят соседи в чате. Больше ничего настраивать не нужно.',
              style: TextStyle(color: Colors.black54),
            ),
            const SizedBox(height: 24),
            TextField(
              key: const Key('settings-nick-field'),
              controller: _ctrl,
              enabled: !_busy,
              decoration: const InputDecoration(
                labelText: 'Ник',
                border: OutlineInputBorder(),
              ),
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _save(),
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!, style: const TextStyle(color: Colors.red)),
            ],
            const SizedBox(height: 24),
            FilledButton(
              key: const Key('settings-nick-save'),
              onPressed: _busy ? null : _save,
              child: const Text('Сохранить'),
            ),
          ],
        ),
      ),
    );
  }
}
