import 'package:flutter/material.dart';

import '../theme/dudka_theme.dart';

/// DESIGN.md step-row progress: 4 pads, red→orange→yellow→white (P069).
int stepProgressLitPads(int percent) {
  if (percent <= 0) return 0;
  if (percent >= 75) return 4;
  if (percent >= 50) return 3;
  if (percent >= 25) return 2;
  return 1;
}

Color stepPadColor(int index) {
  switch (index) {
    case 0:
      return DudkaColors.stepRed;
    case 1:
      return DudkaColors.stepOrange;
    case 2:
      return DudkaColors.stepYellow;
    case 3:
      return DudkaColors.stepWhite;
    default:
      return DudkaColors.ledIdle;
  }
}

class StepProgress extends StatelessWidget {
  const StepProgress({
    super.key,
    required this.percent,
    this.height = 10,
  });

  final int percent;
  final double height;

  @override
  Widget build(BuildContext context) {
    final lit = stepProgressLitPads(percent);
    return Row(
      key: const Key('step-progress'),
      children: [
        for (var i = 0; i < 4; i++) ...[
          if (i > 0) const SizedBox(width: 3),
          Expanded(
            child: SizedBox(
              key: Key('step-pad-$i'),
              height: height,
              child: ColoredBox(
                color: i < lit ? stepPadColor(i) : DudkaColors.ledIdle,
              ),
            ),
          ),
        ],
      ],
    );
  }
}
