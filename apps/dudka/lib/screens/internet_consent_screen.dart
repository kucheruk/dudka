import 'package:flutter/material.dart';

import '../engine/client.dart';
import '../theme/dudka_theme.dart';

class InternetConsentScreen extends StatefulWidget {
  const InternetConsentScreen({
    super.key,
    required this.client,
    required this.onDone,
  });

  final EngineClient client;
  final Future<void> Function() onDone;

  @override
  State<InternetConsentScreen> createState() => _InternetConsentScreenState();
}

class _InternetConsentScreenState extends State<InternetConsentScreen> {
  bool _busy = false;
  String? _error;

  Future<void> _allow() async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.client.enableInternet();
      await widget.onDone();
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('ДУДКА')),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Text('Как Дудка находит соседей', style: DudkaType.display()),
          const SizedBox(height: 16),
          const Text(
            'Это приложение подключится к сигнальному сервису и STUN Студии '
            'на zamoo.team. Они нужны только для знакомства устройств и '
            'согласования прямого WebRTC-канала.',
          ),
          const SizedBox(height: 12),
          const Text(
            'Сервис увидит ваш публичный IP, служебный ID приложения и '
            'описания соединения. Имя, сообщения, файлы и история на сервер '
            'не отправляются. После знакомства они идут напрямую между '
            'устройствами по зашифрованному каналу. TURN не используется.',
          ),
          const SizedBox(height: 12),
          const Text(
            'Пока приложение запущено, оно держит один WebSocket и отправляет '
            'несколько малых STUN-пакетов. Настройка сохраняется на этом '
            'устройстве.',
          ),
          if (_error != null) ...[
            const SizedBox(height: 12),
            Text(
              _error!,
              style: DudkaType.mono(size: 12, color: DudkaColors.danger),
            ),
          ],
          const SizedBox(height: 24),
          FilledButton(
            key: const Key('internet-consent-allow'),
            onPressed: _busy ? null : _allow,
            child: const Text('РАЗРЕШИТЬ И НАЙТИ СВОИХ'),
          ),
          const SizedBox(height: 8),
          Text(
            'Без разрешения приложение остаётся офлайн.',
            style: DudkaType.mono(size: 12, color: DudkaColors.silkscreenDim),
          ),
        ],
      ),
    );
  }
}
