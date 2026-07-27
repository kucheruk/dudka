import 'package:flutter/material.dart';

import 'engine/client.dart';
import 'screens/me_screen.dart';

/// Flutter shell skeleton (P061): macOS-first desktop target.
class DudkaApp extends StatelessWidget {
  const DudkaApp({
    super.key,
    required this.engineBase,
    this.client,
  });

  final String engineBase;
  final EngineClient? client;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'ДУДКА',
      debugShowCheckedModeBanner: false,
      home: MeScreen(engineBase: engineBase, client: client),
    );
  }
}
