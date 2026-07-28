import 'package:flutter/material.dart';

import '../engine/client.dart';
import '../nick/fallback.dart';
import '../theme/dudka_theme.dart';

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
            Text(
              'Как вас зовут?',
              key: const Key('firstrun-title'),
              style: DudkaType.display(),
            ),
            const SizedBox(height: 8),
            Text(
              'Ник видят соседи в чате. Можно пропустить — подставим имя устройства или случайное.',
              style: DudkaType.mono(size: 13, color: DudkaColors.silkscreenDim),
            ),
            const SizedBox(height: 24),
            TextField(
              key: const Key('nick-field'),
              controller: _ctrl,
              enabled: !_busy,
              style: DudkaType.mono(),
              decoration: const InputDecoration(labelText: 'Ник'),
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _submit(skip: false),
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!,
                  style: DudkaType.mono(size: 12, color: DudkaColors.danger)),
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
