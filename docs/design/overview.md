# ДУДКА — архитектура MVP

Статус: Draft  
Владелец: продукт ДУДКА  
Последнее смысловое изменение: 2026-07-27

Связанные требования: `docs/specs/*`. Визуал: [`../../DESIGN.md`](../../DESIGN.md).

## Цель

Один общий чат на квартирный Wi‑Fi. Открыл клиент → видишь online peers и короткий хвост → шлёшь текст или файл. Без аккаунтов, без интернета в рантайме.

## Компоненты

```
┌─────────────────────────────┐     ┌─────────────────────────────┐
│ Flutter shell (mobile/desk) │     │ Linux TUI (Go)              │
│  UI only                    │     │  UI + embeds engine         │
└─────────────┬───────────────┘     └─────────────┬───────────────┘
              │ loopback only                     │ in-process
              ▼                                   ▼
         ┌────────────────────────────────────────────┐
         │              dudkad (Go engine)            │
         │  discovery · register · chat · tail · files│
         └─────────────────────┬──────────────────────┘
                               │ LAN UDP/TCP :41777
                               ▼
                         other peers
```

Браузерная ветка не дублирует UDP/TCP engine и не требует домашнего сервера:

```
Browser tab ── WSS offer/answer/ICE ── zamoo.team signaling (memory only)
     │
     └──────── WebRTC DataChannel / DTLS ──────── other browser tabs
```

Signaling запускается только после отдельного экрана согласия. Прикладные
сообщения и файлы сигнальный сервис не принимает.

## Discovery (не mDNS)

1. Периодический **UDP broadcast** announce на порт `41777`.
2. Получатель делает **TCP register** (надёжный обмен peer info).
3. Если пусто — **subnet scan** (или кнопка «ИСКАТЬ»).
4. Один SSID / одна L2-сеть = один мир. Client isolation на AP ломает продукт — это ожидаемо и честно показывается.

## Роли

Все узлы равны. Среди online детерминированно выбирается **tail-keeper**: lexicographically minimal `peer_id` в множестве «я + известные peers» (`chat.SelectTailKeeper` / DUD-CHAT-121). Keeper хранит кольцо последних **200** сообщений и отдаёт его по `GET /tail` после register. При уходе peer выпадает из множества — выбор пересчитывается тем же правилом (без гистерезиса в MVP).

## Идентичность

- `peer_id` — UUID локально при первом запуске.
- `display_name`: спросили → имя устройства → «Прилагательное + Животное».

## Сообщения и файлы

- Текст: fan-out по известным online peers (TCP).
- Файл: announce в ленте (имя, size, mime, hash, thumbnail для image/*) → chunk download peer-to-peer с прогрессом и отменой; жёсткого лимита размера нет.

## Flutter ↔ Go

Engine слушает только loopback (или UDS на desktop/Linux). GUI не открывает LAN-сокеты сама. Версия протокола engine↔UI согласована с wire major.

## Фаза 2 (не MVP)

Комнаты без кодов (отдельные «каналы» на той же LAN), когда появится запрос семьи.

## Open decisions

- Лицензирование через Дьяк vs полностью free local tool — DUD-PRD-140 / ROADMAP P095.
- Точный bind Flutter (subprocess vs иное) — ROADMAP P060, критерий: дешевизна.

Бэклог поставки — только [`ROADMAP.md`](../../ROADMAP.md); доски в «Делах» нет.
