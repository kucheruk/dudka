# TUI critique + audit (P046 follow-up)

⚠️ DEGRADED: single-context (child agent; no nested sub-agents). Evidence: owner screenshot + `internal/tui/*` source. Mode: **Operate**. World: DESIGN.md charcoal + BBS-lite.

Date: 2026-07-28 · Target: `internal/tui` / `cmd/dudka`

## Critique (UX)

| Heuristic (Operate) | Score 0–4 | Note |
| --- | --- | --- |
| Visibility of system status | 3 | Status strip present; online count readable |
| Match real world | 2 | Lexicon «ДУНУТЬ» broke trust (fixed → ОТПРАВИТЬ) |
| User control | 3 | Enter / scroll / slash cmds / quit |
| Consistency | 2 | GUI charcoal vs TUI washed white |
| Error prevention | 2 | Help dense; empty alone OK |
| Recognition | 3 | Panels labeled |
| Flexibility | 2 | Narrow terminals cramped |
| Aesthetic & minimalist | 1 | Looked like plain dump, not panel |
| Help users recover | 2 | StatusMsg exists; quiet |
| Documentation | 2 | Footer help long |

**Cognitive load:** footer packs >4 options — keep but dim; primary = compose.

**Strengths:** dual-pane topology; RU empty states; script `-once` preserved.

**P0/P1 issues (from screenshot):**
1. P0 — no charcoal: white bg / black text (color profile / unpainted cells)
2. P0 — «ДУНУТЬ» on compose (lexicon ban)
3. P1 — weak hierarchy (status = body weight)
4. P1 — panels lack tonal separation / hairline grammar
5. P2 — silkscreen labels not spaced; step pads invisible without color

## Audit (TUI-adapted)

| Dimension | Score | Key finding |
| --- | --- | --- |
| Accessibility | 2 | Contrast fails when colors drop; keyboard OK |
| Performance | 4 | 500ms poll fine |
| Responsive (terminal) | 2 | LayoutFor mins OK; unpainted gaps on resize |
| Theming | 1 | Tokens exist but not forced to terminal |
| Implementation integrity | 2 | DESIGN tokens in code; runtime washout |
| **Total** | **11/20** | Acceptable → poor until color forced |

Detector CLI on `internal/tui`: clean `[]` (Go/TUI out of web detector scope).

## Fixes shipped this pass

- Force `termenv.TrueColor` (+ `COLORTERM`) in TUI init / RunInteractive
- Full-canvas charcoal paint; tonal peers/feed; LED «ОТПРАВИТЬ»
- Ban «дунуть» in render tests; rebuild `dist/dudka` for owner Terminal
