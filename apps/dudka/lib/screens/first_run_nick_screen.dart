import 'package:flutter/material.dart';

import '../engine/client.dart';
import '../nick/fallback.dart';

/// First-run: единственный обязательный шаг — ник (RU), затем чат (P062 / DUD-UI-110).
class FirstRunNickScreen extends StatefulWidget {
  const FirstRunNickScreen({
    super.key,
    required this.client,
    required this.onDone,
    this.suggested = '',
    this.hostnameForFallback,
    this.nickPick,
  });

  final EngineClient client;
  final Future<void> Function(String nick) onDone;
  final String suggested;
  final String Function()? hostnameForFallback;
  final NickPick? nickPick;

  @override
  State<FirstRunNickScreen> createState() => _FirstRunNickScreenState();
}

class _FirstRunNickScreenState extends State<FirstRunNickScreen> {
  late final TextEditingController _ctrl;
  String? _error;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: widget.suggested);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  Future<void> _submit({required bool skip}) async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final nick = resolveNickFallback(
        typed: skip ? '' : _ctrl.text,
        hostname: widget.hostnameForFallback?.call(),
        pick: widget.nickPick,
      );
      await widget.client.setNick(nick);
      await widget.onDone(nick);
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
      appBar: AppBar(title: const Text('ДУДКА')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(
              'Как вас зовут?',
              key: Key('firstrun-title'),
              style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            const Text(
              'Ник видят соседи в чате. Можно пропустить — подставим имя устройства или случайное.',
              style: TextStyle(color: Colors.black54),
            ),
            const SizedBox(height: 24),
            TextField(
              key: const Key('nick-field'),
              controller: _ctrl,
              enabled: !_busy,
              decoration: const InputDecoration(
                labelText: 'Ник',
                border: OutlineInputBorder(),
              ),
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _submit(skip: false),
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!, style: const TextStyle(color: Colors.red)),
            ],
            const SizedBox(height: 24),
            FilledButton(
              key: const Key('nick-continue'),
              onPressed: _busy ? null : () => _submit(skip: false),
              child: const Text('Продолжить'),
            ),
            const SizedBox(height: 8),
            TextButton(
              key: const Key('nick-skip'),
              onPressed: _busy ? null : () => _submit(skip: true),
              child: const Text('Пропустить'),
            ),
          ],
        ),
      ),
    );
  }
}
