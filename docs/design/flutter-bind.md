# ADR: Flutter ↔ dudkad bind (P060)

Статус: Accepted  
Дата: 2026-07-27  
Владелец: ДУДКА

## Контекст

GUI (Flutter) и engine (`dudkad`) должны жить на одном устройстве. Протокол LAN уже в Go; дублировать его во Flutter нельзя (`AGENTS.md`). Нужен самый дешёвый bind для spike и дальнейшей Фазы 3.

## Решение

**Subprocess + HTTP loopback** (`127.0.0.1`).

1. Flutter (или launcher) поднимает `dudkad` как дочерний процесс с `-listen 127.0.0.1:<port>` и локальным `-data-dir`.
2. UI ходит только в loopback API (`GET /me`, …) — тот же контракт, что у TUI `dudka`.
3. Протокол discovery/chat/files остаётся в Go; Flutter — thin client.

## Альтернативы (отложены)

| Вариант | Почему не сейчас |
|---|---|
| UDS вместо TCP loopback | Чуть меньше порта, но больше платформенного кода; TCP уже работает у TUI |
| FFI / gomobile / platform channel в Go | Дороже DX, сложнее CI и hot-reload |
| Встроить engine в isolate через cgo | Тяжело и ломко для spike |

## Последствия

- Один бинарь engine на платформу + Flutter shell.
- Spike-доказательство: экран показывает ответ `GET /me` (`peer_id`, `name`).
- **P061 first target: macOS desktop** (дешевле iOS/Android на агентских машинах: уже есть `flutter devices` → macos, без эмулятора/подписи). Skeleton: `apps/dudka` (`DudkaApp` + `MeScreen` + `EngineHost` spawn).

## Non-goals этого ADR

- Дизайн UI (`DESIGN.md`) и first-run.
- Автозапуск engine как system service.
- Удалённый engine за пределами устройства.
- iOS/Android targets в P061 (добавятся когда macOS skeleton стабилен).
