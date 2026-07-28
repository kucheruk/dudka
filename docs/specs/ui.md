# DUD-UI — Интерфейс

Статус документа: Draft  
Владелец: ДУДКА  
Последнее смысловое изменение: 2026-07-27

## Назначение и границы

Operate-UI для Flutter GUI на всех поддерживаемых desktop/mobile-платформах и
дополнительного TUI в грамматике [`DESIGN.md`](../../DESIGN.md): step-row
panel × BBS-lite.

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

Мини-настройки GUI содержат смену ника и на desktop один переключатель
«Запускать при входе в систему». Нет аватара, email, телефона, пароля и прочих
полей.

Проверка:

- вход в настройки с чата; сохранение → `POST /nick`; чат показывает новый ник; *(P066: `SettingsNickScreen`)*
- на macOS, Windows и Linux переключатель включает и выключает пользовательский
  автозапуск; при автозапуске приложение начинает работу скрытым в трее;
- evidence: `apps/dudka/test/settings_nick_test.dart`, `./scripts/flutter_settings_test.sh` (P066).

Зависимости: DUD-CHAT-111  
ADR: не требуется

### DUD-UI-120

Priority: P0  
Status: Accepted

Состояния `alone` и `no_network` показываются разным copy на русском:
«БОЛЬШЕ НИКОГО РЯДОМ» vs «НЕТ СЕТИ»; в status strip — «один» / «нет сети».
Текущий пользователь всегда показан первым в списке соседей с меткой «ВЫ» и
включён в `онлайн N`, поэтому открытое приложение не показывает `онлайн 0`.
В `alone` доступна команда «ИСКАТЬ» (скан подсети).

Проверка:

- оба состояния различимы без чтения логов; *(P044 TUI + P065 Flutter status/peers)*
- при пустом `/peers` GUI показывает `{me.name} · ВЫ` и `онлайн 1`;
- при двух удалённых peers GUI показывает текущего пользователя и `онлайн 3`;
- в `alone` явная команда `/search` («ИСКАТЬ») → `POST /scan`, алиасы
  `/scan` и `/поиск`; обычные буквы `S`/`s` не перехватываются, в
  `no_network` действия поиска нет; *(P065, P158)*
- результат действия показывается отдельно от постоянной строки управления;
  технические подробности ошибки пишутся в пользовательский `tui.log`; *(P158)*
- macOS содержит системное объяснение доступа к локальной сети; при первом
  запросе пользователь получает штатный prompt, а не молчаливый сетевой отказ;
  *(P158)*
- evidence: `internal/tui/network_test.go`, `apps/dudka/test/network_seek_test.dart`, `./scripts/flutter_seek_test.sh` (P065).

Зависимости: DUD-NET-140  
ADR: не требуется

### DUD-UI-121

Priority: P0
Status: Accepted

Диагностический пакет Linux TUI можно скопировать одной клавишей без установки
системных пакетов. Пакет предназначен для передачи агенту, который будет
исправлять ошибку.

Проверка:

- пока ошибки нет, F5 ничего не делает и не занимает место в подсказках;
- после ошибки видна подсказка `F5 · КОПИРОВАТЬ ДИАГНОСТИКУ`;
- F5 передаёт в системный буфер терминала через OSC 52: версию, ОС/архитектуру,
  класс адреса движка, состояние сети и счётчики, время и полный текст ошибки,
  а также не более 12 последних строк / 6 КиБ пользовательского `tui.log`;
- пакет не содержит текст сообщений, имена и идентификаторы соседей или точный
  нестандартный адрес движка; управляющие символы удаляются;
- короткий текст ошибки остаётся читаемым и не смешивается с подробностями;
- после F5 интерфейс подтверждает действие;
- evidence: `internal/tui/clipboard_test.go`,
  `internal/tui/screen_test.go` *(P159, P161)*.

Зависимости: DUD-UI-120
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
- *(P040/P148: status strip + self/remote peers; без удалённых →
  `{me} · ВЫ` + «БОЛЬШЕ НИКОГО РЯДОМ»; `dudka -engine` / `internal/tui`)*
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

### DUD-UI-180

Priority: P0
Status: Accepted

Пользовательская desktop-поставка на macOS и Windows открывает Flutter GUI.
Windows публикуется одним переносимым ZIP без установщика:
пользователь распаковывает архив и запускает один `dudka.exe` без терминального
окна. `dudkad.exe` лежит в служебной подпапке и самостоятельно пользователю не
предлагается. Приложение не требует прав администратора и не создаёт ярлыки,
записи удаления или системную установку. Linux публикуется как полноценный TUI
вместе с локальным движком и не требует графической среды.

Проверка:

- Windows portable ZIP → распаковка → `dudka.exe` открывает графическое окно
  без консоли и без установки;
- в пользовательском блоке загрузки нет пары сырых `dudka.exe`/`dudkad.exe`;
- в корне распакованного архива только один пользовательский EXE, движок лежит
  в `internal/`;
- Linux TUI открывает тот же чат, ленту, соседей и файловые команды;
- evidence: Windows/Linux CI build, smoke matrix, содержимое release assets.

Зависимости: DUD-PRD-102
ADR: не требуется

### DUD-UI-181

Priority: P0
Status: Accepted

На macOS, Windows и Linux закрытие главного окна скрывает Дудку в системный
трей. Нажатие на иконку или пункт «Открыть Дудку» возвращает и активирует окно;
полное завершение выполняется явным пунктом «Выйти». В настройках есть
переключатель автозапуска. Автозапущенная Дудка начинает работу скрытой, но
движок остаётся доступен в LAN.

Проверка:

- крестик не завершает процесс, повторное открытие из трея сохраняет сессию;
- меню трея содержит «Открыть Дудку» и «Выйти»;
- явный выход останавливает встроенный engine;
- состояние автозапуска читается из ОС и изменяется без прав администратора;
- evidence: unit/widget tests и platform smoke.

Зависимости: DUD-UI-180
ADR: не требуется

### DUD-UI-182

Priority: P0
Status: Accepted

Иконка desktop-приложения показывает красный кружок с числом новых сообщений,
полученных от других peers, пока окно скрыто или неактивно. Исторический хвост
при первом подключении и собственные сообщения не увеличивают число.
Активация окна обнуляет счётчик. macOS использует badge Dock, Windows —
overlay taskbar, Linux — launcher count там, где среда поддерживает
`com.canonical.Unity.LauncherEntry`; меню/подсказка трея всегда содержит число.

Проверка:

- первый snapshot не создаёт ложных непрочитанных;
- два новых чужих сообщения при скрытом окне дают `2`;
- собственное сообщение не меняет счётчик;
- открытие окна очищает badge;
- evidence: unit tests счётчика и platform smoke.

Зависимости: DUD-UI-181
ADR: не требуется

### DUD-UI-183

Priority: P0
Status: Accepted

Linux TUI поставляется проверяемым shell-однострочником и переносным `.tar.gz`.
Однострочник скачивает точный версионный архив по HTTPS, проверяет встроенный
SHA-256 и устанавливает `dudka` и `dudkad` в пользовательский каталог. Он не
запускает системный пакетный менеджер. Если UFW активен, тот же однострочник
явно объясняет причину, один раз запрашивает sudo и открывает `41777/tcp+udp`
только для автоматически определённой приватной LAN-подсети. Основная
Linux-проверка на Ubuntu CI дополнительно доказывает, что список системных
пакетов не изменился.

Проверка:

- `curl -fsSL https://zamoo.team/dudka/install.sh | sh` устанавливает TUI;
- неверная контрольная сумма останавливает установку;
- на Ubuntu с активным UFW после одной установки сосед может подключиться без
  ручной настройки портов; правило не шире текущей RFC1918-подсети;
- `.tar.gz` остаётся в том же релизе;
- evidence: `packaging/linux/install.sh.in`,
  `./scripts/build_linux_tui_test.sh`, Ubuntu desktop workflow.

Зависимости: DUD-UI-180
ADR: не требуется
