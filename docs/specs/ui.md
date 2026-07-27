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
Status: Draft

Главный экран после first-run — чат: status strip, индикация online peers, лента, compose. Нет обязательных промежуточных «welcome» с маркетингом.

Проверка:

- cold start со сохранённым ником → чат ≤ заявленного NFR;
- evidence: скрин + тайминг.

Зависимости: DUD-PRD-110  
ADR: не требуется

### DUD-UI-110

Priority: P0  
Status: Draft

First-run: единственное обязательное действие — ввод или подтверждение ника (с fallbacks из DUD-CHAT-110). Затем сразу DUD-UI-101.

Проверка:

- нет второго обязательного шага (аватар, телефон, email);
- evidence: flow test.

Зависимости: DUD-CHAT-110  
ADR: не требуется

### DUD-UI-120

Priority: P0  
Status: Draft

Состояния `alone` и `no_network` показываются разным copy на русском: «НИКОГО РЯДОМ» vs отсутствие сети. В `alone` доступна команда «ИСКАТЬ» (subnet scan).

Проверка:

- оба состояния различимы без чтения логов;
- evidence: UI fixtures.

Зависимости: DUD-NET-140  
ADR: не требуется

### DUD-UI-130

Priority: P1  
Status: Draft

Прогресс передачи файла отображается в духе step-row (доля/шаги), согласованно с DESIGN.md; отмена доступна одним жестом/кнопкой рядом с прогрессом.

Проверка:

- во время передачи виден прогресс и Cancel;
- evidence: скрин/видео.

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
