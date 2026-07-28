import 'package:dudka/theme/dudka_theme.dart';
import 'package:dudka/widgets/step_progress.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('DESIGN.md color tokens', () {
    expect(DudkaColors.panel, const Color(0xFF1A1A1A));
    expect(DudkaColors.panelDeep, const Color(0xFF0E0E0E));
    expect(DudkaColors.silkscreen, const Color(0xFFF2F2F2));
    expect(DudkaColors.silkscreenDim, const Color(0xFF8A8A8A));
    expect(DudkaColors.ledActive, const Color(0xFFFF4500));
    expect(DudkaColors.stepRed, const Color(0xFFFF3B30));
    expect(DudkaColors.stepOrange, const Color(0xFFFF9A00));
    expect(DudkaColors.stepYellow, const Color(0xFFFFD600));
    expect(DudkaColors.stepWhite, const Color(0xFFF2F2F2));
  });

  test('theme uses mono + charcoal panel, no CRT fluff', () {
    final t = buildDudkaTheme();
    expect(t.brightness, Brightness.dark);
    expect(t.scaffoldBackgroundColor, DudkaColors.panel);
    expect(t.colorScheme.primary, DudkaColors.ledActive);
    expect(t.textTheme.bodyMedium?.fontFamily, contains('Mono'));
    // Anti CRT / SaaS purple.
    expect(t.colorScheme.primary, isNot(const Color(0xFF6750A4)));
  });

  test('stepProgressLitPads maps percent to 0..4', () {
    expect(stepProgressLitPads(0), 0);
    expect(stepProgressLitPads(1), 1);
    expect(stepProgressLitPads(24), 1);
    expect(stepProgressLitPads(25), 2);
    expect(stepProgressLitPads(50), 3);
    expect(stepProgressLitPads(75), 4);
    expect(stepProgressLitPads(100), 4);
    expect(stepProgressLitPads(200), 4);
    expect(stepProgressLitPads(-1), 0);
  });

  testWidgets('StepProgress lights pads by quarter', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: buildDudkaTheme(),
        home: const Scaffold(
          body: StepProgress(percent: 40, key: Key('steps')),
        ),
      ),
    );
    expect(find.byKey(const Key('step-pad-0')), findsOneWidget);
    expect(find.byKey(const Key('step-pad-1')), findsOneWidget);
    expect(find.byKey(const Key('step-pad-2')), findsOneWidget);
    expect(find.byKey(const Key('step-pad-3')), findsOneWidget);
    expect(stepProgressLitPads(40), 2);

    final p0 = tester.widget<ColoredBox>(
      find
          .descendant(
              of: find.byKey(const Key('step-pad-0')),
              matching: find.byType(ColoredBox))
          .first,
    );
    final p2 = tester.widget<ColoredBox>(
      find
          .descendant(
              of: find.byKey(const Key('step-pad-2')),
              matching: find.byType(ColoredBox))
          .first,
    );
    expect(p0.color, DudkaColors.stepRed);
    expect(p2.color, DudkaColors.ledIdle);
  });
}
