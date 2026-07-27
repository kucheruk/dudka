/// Adaptive chat shell breakpoints (P070 / DUD-UI-140).
///
/// Wide (≥ [dudkaWideBreakpoint]): dual-pane peers | feed+compose.
/// Narrow: horizontal peer strip above the feed column.
const double dudkaWideBreakpoint = 700;

bool isWideChatLayout(double width) => width >= dudkaWideBreakpoint;
