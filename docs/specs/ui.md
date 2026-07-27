# DUD-UI — Интерфейс

Статус документа: Draft  
Владелец: ДУДКА  
Последнее смысловое изменение: 2026-07-27

## Назначение и границы

Operate-UI для Flutter GUI и Linux TUI в грамматике [`DESIGN.md`](../../DESIGN.md): step-row panel × BBS-lite.

### Non-goals

- Отдельный визуальный язык для каждой ОС сверх нативных жестов/safe area.
- Light theme MVP.
- CRT/ANSI/scanline-эффекты.

## Требования

### DUD-UI-101

Priority: P0  
Status: Partial

Главный экран после first-run — чат: status strip, индикация online peers, лента, compose. Нет обязательных промежуточных «welcome» с маркетингом.

Проверка:

- cold start со сохранённым ником → чат ≤ заявленного NFR;
- status strip + peers + текстовая лента (wireframe); *(P063: Flutter `ChatScreen` ← `/me` `/peers` `/status` `/messages`; DESIGN step-row — P069)*
- compose «ДУНУТЬ» → `POST /send`; Flutter↔TUI в LAN; *(P064: `EngineClient.sendText`, `./scripts/flutter_send_test.sh`)*
- evidence: `apps/dudka/test/chat_screen_test.dart`, `apps/dudka/test/compose_send_test.dart`, `./scripts/flutter_chat_test.sh`, `./scripts/flutter_send_test.sh` (P063/P064).

Зависимости: DUD-PRD-110  
ADR: не требуется

### DUD-UI-110

Priority: P0  
Status: Accepted

First-run: единственное обязательное действие — ввод или подтверждение ника (с fallbacks из DUD-CHAT-110). Затем сразу DUD-UI-101.

Проверка:

- нет второго обязательного шага (аватар, телефон, email); *(P062: Flutter `FirstRunNickScreen` — только ник + «Продолжить»/«Пропустить»)*
- cold start без подтверждённого ника → RU first-run → чат; с подтверждённым → сразу чат;
- evidence: `apps/dudka/test/first_run_test.dart`, `./scripts/flutter_firstrun_test.sh` (P062).

Зависимости: DUD-CHAT-110  
ADR: не требуется

### DUD-UI-115

Priority: P0  
Status: Accepted

Мини-настройки GUI содержат только смену ника. Нет аватара, email, телефона, пароля и прочих полей.

Проверка:

- вход в настройки с чата; сохранение → `POST /nick`; чат показывает новый ник; *(P066: `SettingsNickScreen`)*
- evidence: `apps/dudka/test/settings_nick_test.dart`, `./scripts/flutter_settings_test.sh` (P066).

Зависимости: DUD-CHAT-111  
ADR: не требуется

### DUD-UI-120

Priority: P0  
Status: Accepted

Состояния `alone` и `no_network` показываются разным copy на русском: «НИКОГО РЯДОМ» vs «НЕТ СЕТИ». В `alone` доступна команда «ИСКАТЬ» (subnet scan).

Проверка:

- оба состояния различимы без чтения логов; *(P044 TUI + P065 Flutter status/peers)*
- в `alone` кнопка «ИСКАТЬ» → `POST /scan`, в `no_network` кнопки нет; *(P065)*
- evidence: `internal/tui/network_test.go`, `apps/dudka/test/network_seek_test.dart`, `./scripts/flutter_seek_test.sh` (P065).

Зависимости: DUD-NET-140  
ADR: не требуется

### DUD-UI-130

Priority: P1  
Status: Draft

Прогресс передачи файла отображается в духе step-row (доля/шаги), согласованно с DESIGN.md; отмена доступна одним жестом/кнопкой рядом с прогрессом.

Проверка:

- во время передачи виден прогресс; *(P052: TUI `NN%` на FILE-строке, `GET /files/transfers`)*
- Cancel; *(P053: `/cancel <file_id>` → `CANCELLED discarded`, не 100%)*
- evidence: UI fixtures / `scripts/file_progress_test.sh`, `scripts/file_cancel_test.sh`.

Зависимости: DUD-FILE-110  
ADR: не требуется

### DUD-UI-140

Priority: P1  
Status: Draft

На широком GUI (≥ 700 dp logical width) используется dual-pane: peers | лента. На узком — peers как strip/лист поверх или над лентой, без обязательного отдельного tab root.

Проверка:

- resize desktop переключает layout без потери compose text;
- evidence: adaptive screenshot set.

Зависимости: DUD-UI-101  
ADR: не требуется

### DUD-UI-150

Priority: P0  
Status: Partial

Linux TUI показывает те же сущности: peers, лента, compose, статусы `alone`/`no_network`, прогресс файла (текстом). Не требует GUI.

Проверка:

- сценарий текст+файл между TUI и Flutter;
- *(P040: status strip + peers; пусто → «НИКОГО РЯДОМ»; `dudka -engine` / `internal/tui`)*
- *(P041: FEED из `GET /messages`, строки `время · ник · текст`)*
- *(P042: compose Enter/`-send` → `POST /send`; два peer обмениваются текстом)*
- *(P043: `/nick Имя` / `-nick` → смена ника; видно в следующих сообщениях)*
- evidence: smoke script.

Зависимости: DUD-PRD-102  
ADR: не требуется

### DUD-UI-160

Priority: P1  
Status: Draft

Визуал GUI следует DESIGN.md: charcoal panel, silkscreen labels, mono stack, step colors; без облачных иллюстраций и «мессенджерных» пузырей как основной язык.

Проверка:

- design review против DESIGN.md Do's/Don'ts;
- evidence: скриншоты в карточке.

Зависимости: нет  
ADR: не требуется
