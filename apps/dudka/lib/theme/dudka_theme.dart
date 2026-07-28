import 'package:flutter/material.dart';

/// DESIGN.md tokens — charcoal panel × silkscreen × step colors (P069 / DUD-UI-160).
abstract final class DudkaColors {
  static const panel = Color(0xFF1A1A1A);
  static const panelDeep = Color(0xFF0E0E0E);
  static const silkscreen = Color(0xFFF2F2F2);
  static const silkscreenDim = Color(0xFF8A8A8A);
  static const stepRed = Color(0xFFFF3B30);
  static const stepOrange = Color(0xFFFF9A00);
  static const stepYellow = Color(0xFFFFD600);
  static const stepWhite = Color(0xFFF2F2F2);
  static const ledActive = Color(0xFFFF4500);
  static const ledIdle = Color(0xFF3A3A3A);
  static const segment = Color(0xFFFF3B30);
  static const danger = Color(0xFFFF3B30);
  static const ok = Color(0xFFFFD600);
}

abstract final class DudkaType {
  static const monoFamily = 'JetBrains Mono';
  static const monoFallback = <String>[
    'UI Monospace',
    'Menlo',
    'Consolas',
    'Courier',
    'monospace'
  ];

  static TextStyle mono({
    double size = 15,
    FontWeight weight = FontWeight.w400,
    Color color = DudkaColors.silkscreen,
    double letterSpacing = 0.01 * 15,
    double height = 1.35,
  }) {
    return TextStyle(
      fontFamily: monoFamily,
      fontFamilyFallback: monoFallback,
      fontSize: size,
      fontWeight: weight,
      color: color,
      letterSpacing: letterSpacing,
      height: height,
    );
  }

  static TextStyle label({Color color = DudkaColors.silkscreenDim}) {
    return mono(
        size: 11,
        weight: FontWeight.w500,
        color: color,
        letterSpacing: 0.12 * 11,
        height: 1.2);
  }

  static TextStyle display({Color color = DudkaColors.silkscreen}) {
    return mono(
        size: 22,
        weight: FontWeight.w700,
        color: color,
        letterSpacing: 0.04 * 22,
        height: 1.15);
  }
}

ThemeData buildDudkaTheme() {
  final base = ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    scaffoldBackgroundColor: DudkaColors.panel,
    canvasColor: DudkaColors.panel,
    dividerColor: DudkaColors.silkscreenDim.withValues(alpha: 0.4),
    colorScheme: const ColorScheme.dark(
      surface: DudkaColors.panel,
      primary: DudkaColors.ledActive,
      onPrimary: DudkaColors.panelDeep,
      secondary: DudkaColors.stepYellow,
      onSecondary: DudkaColors.panelDeep,
      error: DudkaColors.danger,
      onSurface: DudkaColors.silkscreen,
      outline: DudkaColors.silkscreenDim,
    ),
  );

  final body = DudkaType.mono();
  final label = DudkaType.label();

  return base.copyWith(
    textTheme: TextTheme(
      bodyLarge: body.copyWith(fontSize: 16),
      bodyMedium: body,
      bodySmall: body.copyWith(fontSize: 13, color: DudkaColors.silkscreenDim),
      titleLarge: DudkaType.display(),
      titleMedium: DudkaType.mono(size: 16, weight: FontWeight.w700),
      labelLarge: label.copyWith(fontSize: 12, letterSpacing: 1.4),
      labelMedium: label,
      labelSmall: label.copyWith(fontSize: 10),
    ),
    appBarTheme: AppBarTheme(
      backgroundColor: DudkaColors.panelDeep,
      foregroundColor: DudkaColors.silkscreen,
      elevation: 0,
      scrolledUnderElevation: 0,
      titleTextStyle:
          DudkaType.mono(size: 16, weight: FontWeight.w700, letterSpacing: 2),
      iconTheme: const IconThemeData(color: DudkaColors.silkscreen),
      shape: const Border(
        bottom: BorderSide(color: DudkaColors.silkscreenDim, width: 1),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: DudkaColors.panelDeep,
      hintStyle: DudkaType.mono(size: 14, color: DudkaColors.silkscreenDim),
      labelStyle: DudkaType.label(),
      enabledBorder: const OutlineInputBorder(
        borderRadius: BorderRadius.zero,
        borderSide: BorderSide(color: DudkaColors.silkscreenDim, width: 1),
      ),
      focusedBorder: const OutlineInputBorder(
        borderRadius: BorderRadius.zero,
        borderSide: BorderSide(color: DudkaColors.ledActive, width: 1),
      ),
      border: const OutlineInputBorder(
        borderRadius: BorderRadius.zero,
        borderSide: BorderSide(color: DudkaColors.silkscreenDim, width: 1),
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: DudkaColors.ledActive,
        foregroundColor: DudkaColors.panelDeep,
        disabledBackgroundColor: DudkaColors.ledIdle,
        disabledForegroundColor: DudkaColors.silkscreenDim,
        shape: const RoundedRectangleBorder(borderRadius: BorderRadius.zero),
        textStyle: DudkaType.mono(
            size: 13,
            weight: FontWeight.w700,
            letterSpacing: 1.5,
            color: DudkaColors.panelDeep),
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: DudkaColors.silkscreen,
        side: const BorderSide(color: DudkaColors.silkscreenDim, width: 1),
        shape: const RoundedRectangleBorder(borderRadius: BorderRadius.zero),
        textStyle: DudkaType.mono(
            size: 12, weight: FontWeight.w600, letterSpacing: 1.5),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: DudkaColors.silkscreen,
        textStyle: DudkaType.mono(
            size: 12, weight: FontWeight.w600, letterSpacing: 1.2),
        shape: const RoundedRectangleBorder(borderRadius: BorderRadius.zero),
      ),
    ),
    progressIndicatorTheme: const ProgressIndicatorThemeData(
      color: DudkaColors.ledActive,
      circularTrackColor: DudkaColors.ledIdle,
    ),
  );
}
