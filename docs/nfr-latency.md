# NFR latency vs DUD-PRD-120 (P090)

Status: Measured (lab) · 2026-07-28  
Spec: `DUD-PRD-120` — p95 text ≤ 500 ms; peer appear ≤ 3 s; GUI interactive p75 ≤ 2 s (phone).

## Стенд

| Роль | Устройство | Сеть |
| --- | --- | --- |
| A | macOS arm64 (agent host) | loopback + same host LAN iface |
| B | second `dudkad` process (same host) | loopback announce unicast / shared SO_REUSEPORT |
| C | — | отдельный телефон не подключён к агенту (gap ниже) |

Метод: `./scripts/nfr_latency_test.sh` — два peer, 30 текстовых send, p50/p95 RTT по появлению в `/messages`.

## Таблица (lab, same-host)

| Метрика | Цель | Замер | Вердикт |
| --- | --- | --- | --- |
| text peer→peer p95 | ≤ 500 ms | **9.0 ms** (n=30, same-host, 2026-07-28) | ✅ lab |
| peer appear after announce p95 | ≤ 3 s | peers visible &lt; 3 s in nfr script wait loop | ✅ lab |
| GUI start → interactive feed p75 | ≤ 2 s (phone) | macOS `.app` cold start ~1–2 s qualitative | ⚠️ phone не замерен |

## Gap

- Нет третьего устройства (телефон) на агентском стенде → phone p75 GUI **не закрыт замером**.
- Same-host ≠ квартирный Wi‑Fi; домашний роутер — P093 / `docs/platform-smoke.md`.

Спека `DUD-PRD-120` остаётся P1; lab evidence достаточна для ROADMAP P090 с явным phone gap.
