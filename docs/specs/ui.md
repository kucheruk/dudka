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
- compose «ОТПРАВИТЬ» → `POST /send`; Flutter↔TUI в LAN; *(P064: `EngineClient.sendText`, `./scripts/flutter_send_test.sh`; лексикон: не «дунуть»)*
- Flutter↔Flutter текст+файл на двух peers; *(P071: два `dudkad` + `live_send`/`live_wait_text`/`live_announce`/`live_fetch`, `./scripts/flutter_ff_test.sh`)*
- evidence: `apps/dudka/test/chat_screen_test.dart`, `apps/dudka/test/compose_send_test.dart`, `./scripts/flutter_chat_test.sh`, `./scripts/flutter_send_test.sh`, `./scripts/flutter_ff_test.sh` (P063/P064/P071).

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

Состояния `alone` и `no_network` показываются разным copy на русском: «НИКОГО РЯДОМ» vs «НЕТ СЕТИ»; в status strip — «один» / «нет сети». В `alone` доступна команда «ИСКАТЬ» (скан подсети).

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

- во время передачи виден прогресс; *(P052: TUI `NN%`; P067 Flutter `%`; P069 step-row pads)*
- Cancel; *(P053 / P067 Flutter «ОТМЕНА»)*
- evidence: UI fixtures / `scripts/file_progress_test.sh`, `scripts/file_cancel_test.sh`, `./scripts/flutter_files_test.sh`, `./scripts/flutter_theme_test.sh` (P067/P069).

Зависимости: DUD-FILE-110  
ADR: не требуется

### DUD-UI-140

Priority: P1  
Status: Accepted

На широком GUI (≥ 700 dp logical width) используется dual-pane: peers | лента. На узком — peers как strip/лист поверх или над лентой, без обязательного отдельного tab root.

Проверка:

- resize desktop переключает layout без потери compose text; *(P070: один `_compose` controller переживает narrow↔wide)*
- evidence: `apps/dudka/test/layout_test.dart`, `./scripts/flutter_layout_test.sh` (P070).

Зависимости: DUD-UI-101  
ADR: не требуется

### DUD-UI-150

Priority: P0  
Status: Partial

Linux TUI показывает те же сущности: соседи, лента, compose, статусы `alone`/`no_network`, прогресс файла (текстом). Не требует GUI. User-facing строки — русский (`DUD-PRD-103`).

Проверка:

- сценарий текст+файл между TUI и Flutter;
- *(P040: status strip + peers; пусто → «НИКОГО РЯДОМ»; `dudka -engine` / `internal/tui`)*
- *(P041: ЛЕНТА из `GET /messages`, строки `время · ник · текст`)*
- *(P042: compose Enter/`-send` → `POST /send`; два peer обмениваются текстом)*
- *(P043: `/nick Имя` / `-nick` → смена ника; видно в следующих сообщениях)*
- *(P072: заголовки СОСЕДИ/ЛЕНТА/ВВОД, «онлайн N», «ФАЙЛ …»)*
- evidence: smoke script; `./scripts/ru_ui_test.sh` (P072).

Зависимости: DUD-PRD-102  
ADR: не требуется

### DUD-UI-160

Priority: P1  
Status: Accepted

Визуал GUI следует DESIGN.md: charcoal panel, silkscreen labels, mono stack, step colors; без облачных иллюстраций и «мессенджерных» пузырей как основной язык.

Проверка:

- design review против DESIGN.md Do's/Don'ts; *(P069: `buildDudkaTheme`, mono + charcoal, без CRT)*
- file progress как step-row (4 pads); *(P069: `StepProgress`)*
- evidence: `apps/dudka/test/theme_test.dart`, `./scripts/flutter_theme_test.sh` (P069).

Зависимости: нет  
ADR: не требуется

### DUD-UI-170

Priority: P0
Status: Accepted

Attach открывает нативный системный выбор файлов. Один или несколько выбранных
файлов добавляются в черновик compose и не публикуются до явной отправки.
Черновик показывает thumbnail для изображения или файловую метку, имя и
действие удаления. Текст compose отправляется как комментарий перед
file-announce; разрешена отправка только файлов.

Проверка:

- attach не требует ручного ввода пути и поддерживает multi-select;
- после выбора `POST /files/announce` ещё не вызван;
- send публикует текст и все оставшиеся вложения, затем очищает их;
- evidence: `apps/dudka/test/file_transfer_test.dart`, macOS build.

Зависимости: DUD-FILE-101
ADR: не требуется

### DUD-UI-171

Priority: P0
Status: Accepted

Главный экран использует один компактный хедер со status и настройками.
Текст ленты выделяется и копируется системными средствами. Enter вставляет
перенос строки; Cmd+Enter на macOS и Ctrl+Enter на Windows/Linux отправляют.
Attach и send показаны иконками скрепки и самолётика с русскими tooltip.

Проверка:

- нет отдельного AppBar «Чат» над status strip;
- лента находится внутри selection-контейнера;
- compose многострочный и имеет оба keyboard shortcut;
- evidence: `apps/dudka/test/chat_screen_test.dart`,
  `apps/dudka/test/compose_send_test.dart`.

Зависимости: DUD-UI-101
ADR: не требуется

### DUD-UI-172

Priority: P1
Status: Accepted

Иконка приложения следует миру rhythm-machine panel, но не рисует буквальную
дудку, раструб или speech bubble. Знак — четыре сцепленных тяжёлых блока:
красный сверху-слева, оранжевый сверху-справа, off-white снизу-слева и жёлтый
снизу-справа. Их срезы образуют ломаный чёрный центр на charcoal-поле.
Геометрия угловатая и читается в 16 px. Утверждённый PNG-источник порождает
нативные размеры macOS, iOS, Android и Windows; дефолтный Flutter знак не
используется.

Проверка:

- asset catalogs/resources всех четырёх GUI-платформ содержат новый знак;
- утверждённый источник: `apps/dudka/assets/branding/app_icon_source.png`;
- нормализованный мастер: `apps/dudka/assets/branding/app_icon_master.png`;
- генерация: `./scripts/generate_app_icons.sh`;
- evidence: platform builds и визуальная проверка 1024/16 px.

Зависимости: DUD-UI-160
ADR: не требуется
