# DUD-CHAT — Чат, идентичность, хвост

Статус документа: Draft  
Владелец: ДУДКА  
Последнее смысловое изменение: 2026-07-27

## Назначение и границы

Общий чат квартиры: ник, текст, хвост истории у online. Файлы — в DUD-FILE.

### Non-goals

- Несколько комнат в MVP.
- Серверная персистентность вне устройств peers.
- Редактирование/удаление чужой истории на всех узлах (достаточно локального отображения).

## Требования

### DUD-CHAT-101

Priority: P0  
Status: Partial

Текстовое сообщение имеет: `msg_id`, `peer_id`, `display_name_at_send`, `ts` (UTC), `text` (≤ 4000 UTF-8 code points). Отправитель доставляет его всем известным online peers по TCP session.

Проверка:

- три online peer: все трое видят сообщение; *(P030: два peer через `POST /send` → `GET /messages` ≤ 2 s; третий — позже)*
- oversized text отвергается с понятной ошибкой UI; *(P031: `ValidateText` / `ErrTextTooLong`, HTTP 4xx на `POST /send`)*
- evidence: protocol tests.

Зависимости: DUD-NET-111  
ADR: не требуется

### DUD-CHAT-110

Priority: P0  
Status: Draft

При первом запуске клиент создаёт стабильный `peer_id` (UUID) и запрашивает `display_name`. Если пользователь пропускает ввод: берётся имя устройства; если его нельзя получить осмысленно — генерируется «Прилагательное + Животное» из локального словаря.

Проверка:

- три ветки имени покрыты тестами; *(P011 engine + P062 Flutter `nick/fallback.dart` + skip→«Прилагательное+Животное»)*
- `peer_id` переживает рестарт приложения;
- evidence: `apps/dudka/test/nick_fallback_test.dart`, `apps/dudka/test/first_run_test.dart`, `./scripts/flutter_firstrun_test.sh` (P062).

Зависимости: нет  
ADR: не требуется

### DUD-CHAT-111

Priority: P1  
Status: Partial

Смена `display_name` применяется к новым сообщениям и announce; уже показанные сообщения сохраняют `display_name_at_send`.

Проверка:

- после смены ника старые строки в ленте не переписываются; *(P043: TUI `/nick` / `-nick` → `POST /nick`; `display_name_at_send` на новых msg)*
- evidence: UI test / protocol fixture.

Зависимости: DUD-CHAT-110  
ADR: не требуется

### DUD-CHAT-120

Priority: P0  
Status: Accepted

Среди online peers детерминированно выбирается один tail-keeper. Он хранит кольцо последних **200** сообщений (текст + метаданные файловых announce) и отдаёт его новому peer после register.

Проверка:

- новый peer после join видит ≤ 200 последних сообщений, согласованных с keeper; *(P033: TCP `tail_req`/`tail`, loopback `GET /tail`, ring `MaxTailMessages=200`)*
- при уходе keeper новый keeper продолжает отдавать хвост; *(P034: `PeerTTL` prune → `peer_gone` → пересчёт min peer_id; третий peer тянет хвост у нового keeper)*
- evidence: multi-peer integration.

Зависимости: DUD-NET-110  
ADR: не требуется

### DUD-CHAT-121

Priority: P1  
Status: Partial

Алгоритм выбора tail-keeper документирован (например lexicographically minimal `peer_id` среди online) и одинаков на всех клиентах; при равных условиях нет флип-флопа чаще 1 раза / 5 s.

Проверка:

- property/unit тест выбора; *(P032: `SelectTailKeeper` / `SelectTailKeeperAmong`, table-driven наборы id)*
- evidence: тест + короткий абзац в overview.

Зависимости: DUD-CHAT-120  
ADR: не требуется

### DUD-CHAT-130

Priority: P2  
Status: Accepted

MVP допускает best-effort доставку текста без обязательного end-to-end ack; сообщение, не попавшее в хвост и не доставленное offline peer, может быть потеряно. Это отражается в PRODUCT/UI copy («видят те, кто онлайн»).

Проверка:

- copy и спека согласованы;
- нет ложного индикатора «доставлено всем», если нет ack-протокола; *(P035: `POST /send` → только `accepted`/`queued` + поле `queued`; логи `chat_accepted`/`chat_queued`/`chat_fanout_ok`, без delivered)*
- evidence: review copy.

Зависимости: DUD-CHAT-101  
ADR: не требуется
