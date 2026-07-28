# ДУДКА — ROADMAP

Единственный бэклог поставки. Доски в «Делах» **нет** — работа только по этому файлу.  
Детали контракта: [`docs/specs/`](docs/specs/), архитектура: [`docs/design/overview.md`](docs/design/overview.md), UI: [`DESIGN.md`](DESIGN.md).

**Как читать**
- Каждый пункт — маленькая дельта с **видимым результатом** (команда, тест, экран, артефакт).
- `P001`…`P100` — рекомендуемый порядок; меньший номер раньше. Пробелы в нумерации — запас под вставки.
- Отмечай `[x]` в том же PR/коммите, где закрыл результат.
- Приоритеты продукта: дешевизна разработки → удобство UI → скорость.

---

## Фаза 0 — Контур

- [x] P001 Спеки и PRODUCT/DESIGN в `master` (видимый результат: файлы в репо)
- [x] P002 Forgejo `vetinary/dudka` + push `master` (результат: клон по SSH)
- [x] P003 `scripts/check.sh` — `go test ./...` (или no-op success, пока нет кода) + запись в README как локальный гейт
- [x] P004 Корневой `go.mod` модуля `dudka` / `github.com/…` или studio path — `go list ./...` не пустой после P010
- [x] P005 Каркас каталогов `cmd/dudkad`, `internal/…`, `cmd/dudka` (TUI) — дерево видно в репо, сборка stub `main` печатает версию

**Фаза 0 готова:** агент клонирует, `./scripts/check.sh` зелёный на stub.

---

## Фаза 1 — Engine + discovery + текст + TUI

### 1.1 Идентичность и процесс

- [x] P010 Локальный `peer_id` (UUID) создаётся при первом старте и лежит на диске — повторный старт печатает тот же id
- [x] P011 Выбор `display_name`: CLI-флаг / prompt → иначе hostname → иначе «Прилагательное+Животное» — три ветки покрыты unit-тестом
- [x] P012 Процесс `dudkad` стартует, пишет в stdout `ready peer_id=… name=…` и слушает loopback health `GET /health` → 200

### 1.2 Loopback API (ещё без LAN)

- [x] P015 Минимальный loopback JSON: `GET /me` отдаёт peer_id и name — `curl` с 127.0.0.1 работает, с чужого IP отвергается
- [x] P016 `POST /nick` меняет name, следующий `GET /me` показывает новое — старые msg не трогаем (пока msg нет)

### 1.3 Discovery

- [x] P020 UDP broadcast announce на `:41777` раз в N секунд — второй процесс в той же LAN видит пакет (tcpdump/лог `announce_rx`)
- [x] P021 По announce — TCP register: оба peer появляются друг у друга в памяти — `GET /peers` на loopback показывает соседа
- [x] P022 `instance_id` меняется при рестарте — сосед помечает peer «обновился», без дубля двух зомби-записей
- [x] P023 Несовместимый `proto_major` в register — отказ + строка в логе/статусе, сессия не портится
- [x] P024 Subnet scan fallback: команда/endpoint `POST /scan` находит peer при выключенном broadcast (тест: фильтр UDP)
- [x] P025 Guard: попытка dial на публичный IP из конфига не уходит в WAN — тест с фейковым dialer (`DUD-NET-101`)

### 1.4 Текст и хвост

- [x] P030 `POST /send` текста → fan-out всем online — второй peer видит сообщение в `GET /messages` ≤ 2 s локально
- [x] P031 Валидация длины текста (≤ 4000) — oversized → 4xx и понятная ошибка
- [x] P032 Выбор tail-keeper (min peer_id) — unit-тест на наборах id
- [x] P033 Хвост 200 на keeper: третий peer после register получает `GET /tail` согласованный с keeper
- [x] P034 Уход keeper → перевыбор → новый peer всё ещё получает хвост (integration на 3 процессах)
- [x] P035 Best-effort: нет ложного «доставлено всем» в API/логах — только `accepted` / `queued`

### 1.5 Linux TUI v0

- [x] P040 TUI показывает status + список peers (пусто → «НИКОГО РЯДОМ»)
- [x] P041 TUI показывает ленту сообщений из engine
- [x] P042 TUI: ввод строки Enter = send — два TUI в одной сети обмениваются текстом
- [x] P043 TUI: смена ника из меню/команды — видно в следующих сообщениях
- [x] P044 Состояние `no_network` vs `alone` различаются в TUI copy
- [x] P045 `./scripts/check.sh` гоняет protocol tests (2+ peer) — зелёный на CI/локально
- [x] P046 Интерактивный TUI (bubbletea/lipgloss): фиксированные панели status|peers|feed|compose, redraw, DESIGN.md charcoal/silkscreen, дефолт без флагов = TUI (не CLI-дамп); one-shot флаги для скриптов сохранить

**Фаза 1 готова:** два TUI на Wi‑Fi шлют текст и подтягивают хвост; WAN-тест зелёный.

---

## Фаза 2 — Файлы

- [x] P050 File-announce в ленте (имя, size, mime, hash, file_id) без автозагрузки всего файла на всех
- [x] P051 Скачивание чанками у источника — получатель имеет полный файл на диске
- [x] P052 Прогресс 0–100% в API/TUI во время скачивания
- [x] P053 Отмена скачивания — файл не в статусе «успех», частичный discarded
- [x] P054 Warning в TUI при size > 100 MiB до старта, передача всё равно возможна
- [x] P055 Проверка hash после загрузки — битый файл → ошибка
- [x] P056 Thumbnail jpeg/png/webp в announce/ленте TUI (хотя бы путь/ASCII-пометка + файл превью)
- [x] P057 HEIC-превью на платформах где декод есть; иначе честный fallback без фейкового превью
- [x] P058 TUI↔TUI: картинка с превью и произвольный бинарник доходят end-to-end

**Фаза 2 закрыта:** файлы и превью между двумя TUI (P050–P058).

---

## Фаза 3 — Flutter shell

- [x] P060 Spike: самый дешёвый bind Flutter↔`dudkad` (subprocess+loopback или иное) — 1-page ADR/заметка в `docs/design/flutter-bind.md` + hello `GET /me` на экране
- [x] P061 Flutter app skeleton (iOS/Android или desktop — что дешевле первым) запускается и показывает `/me`
- [x] P062 First-run экран ника (RU) → дальше сразу чат
- [x] P063 Экран чата: status strip + peers + лента текста (пока без дизайна step-row — wireframe ок)
- [x] P064 Compose «ОТПРАВИТЬ» шлёт текст — Flutter↔TUI в одной LAN
- [x] P065 Состояния `alone` / `no_network` + кнопка «ИСКАТЬ»
- [x] P066 Мини-настройки: только ник
- [x] P067 Отправка/приём файла из Flutter с прогрессом и отменой
- [x] P068 Превью картинок в ленте Flutter
- [x] P069 Визуал `DESIGN.md`: charcoal panel, silkscreen labels, mono, step-progress (без CRT-фанатизма)
- [x] P070 Wide layout dual-pane; narrow — peer strip (resize desktop не теряет текст compose)
- [x] P071 Flutter↔Flutter текст+файл на двух устройствах
- [x] P072 Все user-facing строки GUI — русский (`DUD-PRD-103`)

**Фаза 3 готова:** GUI и TUI вместе живут в квартире.

---

## Фаза 4 — Упаковка

- [x] P080 Скрипт/док: сборка Linux TUI-бинаря одной командой → артефакт в `dist/`
- [x] P081 Сборка macOS desktop (Flutter) → открывается `.app` / архив
- [x] P082 Сборка Windows desktop → запускаемый артефакт (`./scripts/build_windows_app.sh` → `dist/dudkad-windows-*.exe` + TUI; Flutter GUI на Windows-хосте — `dist/BUILD-WINDOWS.md`)
- [x] P083 Сборка Android APK/AAB для sidecar — ставится на телефон семьи (`./scripts/build_android_apk.sh` → `dist/dudka-android.apk`/`.aab`; engine sidecar note in `BUILD-ANDROID.md`)
- [x] P084 iOS: ad-hoc или TestFlight path задокументирован; билд хотя бы на одном устройстве (`docs/build-ios.md`, `./scripts/build_ios_app.sh` → unsigned iphoneos Runner.app; physical install needs Apple signing)
- [x] P085 README «как поставить семье за 5 минут» без аккаунтов ([`docs/family-install.md`](docs/family-install.md))
- [x] P086 Смоук-таблица платформ в `docs/` (что проверено руками) — обновляется при каждой поставке ([`docs/platform-smoke.md`](docs/platform-smoke.md))

**Фаза 4 готова:** семья ставит и шлёт строку/файл без аккаунта.

---

## Фаза 4b — Лендинг на zamoo.team

Статика: репозиторий [`~/zt/zamoo.team/`](../zamoo.team/) (отдельный git; правки напрямую в HTML/CSS). Цель: страница Дудки + ссылка с главной.

- [x] P120 Копирайт/outline страницы Дудки (1 screen: что это, LAN-only, без аккаунтов) — [`docs/landing-zamoo.md`](docs/landing-zamoo.md)
- [x] P121 HTML-страница `zamoo.team/dudka.html` в стиле сайта: герой + блоки ОС (плейсхолдеры)
- [x] P122 Ссылка «Дудка» в футере главной `zamoo.team/index.html`
- [x] P123 Живые скриншоты + ссылки на установщики (macOS/Windows/Linux/Android/iOS) — когда артефакты фазы 4 готовы (`zamoo.team/dudka.html` + `assets/dudka/*`; install → build scripts; DESIGN stills — Screen Recording TCC на агентском Mac)
- [x] P124 Oneliner установки семьи (док + блок на странице), когда есть стабильные артефакты из P080–P085 (`docs/family-install.md` + блок `#oneliner-title` на `dudka.html`)

**Фаза 4b готова:** с zamoo.team можно понять Дудку и скачать/поставить без аккаунта.

---

## Фаза 5 — Полировка

- [x] P090 Замер NFR текста на 2–3 устройствах — таблица vs `DUD-PRD-120` (или явный gap + правка спеки) (`docs/nfr-latency.md`, `./scripts/nfr_latency_test.sh`; phone gap explicit)
- [x] P091 UX при занятом порте `:41777` — процесс жив, статус понятен (TCP/UDP fallback + `port_note` в `/status`/TUI/GUI)
- [x] P092 UX proto mismatch — «обнови Дудку», без краша (status/TUI/GUI copy)
- [x] P093 Прогон discovery на домашнем роутере; баги → фикс или papercut (lab LAN + note в `platform-smoke.md`)
- [x] P094 Удалить временные костыли bind/discovery с пометкой в коммите (временных bind-костылей не осталось; Flutter subprocess bind — постоянный ADR)
- [x] P095 Решение `DUD-PRD-140` (лицензия/Community) записано в спеке Accepted или Withdrawn с причиной (Community, no callhome)
- [x] P096 Нет callhome/update-check в бинарнике — grep/тест на отсутствие WAN clients (`./scripts/no_callhome_test.sh`)

**Фаза 5 готова:** MVP можно отдавать семье без стыда за диагноз сети.

---

## Фаза 6 — После MVP

- [x] P097 Комнаты без кодов: список каналов на той же LAN (спека + минимальный UI) (`docs/specs/rooms.md`, `/channels`, TUI КАНАЛЫ)
- [x] P098 Опциональный ack/retry текста, если best-effort бесит на практике (`want_ack` → type `ack`)
- [x] P099 Offline-раздача обновлений по LAN (без cloud) (`/updates`, `docs/updates-lan.md`)
- [x] P100 Store-пакеты (если понадобятся) — только после P095 и без runtime-интернета (Withdrawn: Community/no-store после P095; store не нужен)

---

## Фаза 7 — Домашние агенты (MCP на LAN)

Контракт: [`docs/specs/agents.md`](docs/specs/agents.md) (`DUD-AGT-*`). Не Диалоги/Доверие; только квартирная сеть.

- [x] P110 Спека MCP home-agents Accepted: surface tools, no-WAN, тройной префикс ника `{agent}·{model}·{host}` (`DUD-AGT-101/110`)
- [x] P111 Advertise/discovery агента в LAN (peer + agent marker; тот же no-WAN контур) (`is_agent`, `-agent`)
- [x] P112 Нормализация/валидация ника агента с обязательным тройным префиксом; человек без такого префикса (`internal/agent`)
- [x] P113 MCP tool: agent → чат (send текста в общий feed) (`dudka_send` in `cmd/dudka-mcp`)
- [x] P114 MCP tool: чат → agent (получить входящий текст) (`dudka_inbox`)
- [x] P115 Smoke: два процесса (человек + agent stub) на LAN/loopback обмениваются текстом; тройной префикс виден в ленте (`./scripts/agent_mcp_smoke.sh`)
- [x] P116 Агентский skill + публичная инструкция MCP и модель source-available лицензии (`.skills/dudka/SKILL.md`, `docs/licensing.md`)

**Фаза 7 готова:** домашний агент в том же чате квартиры, отличим по нику, без интернета.

---

## Фаза 8 — Desktop UX вложений и чата

- [x] P130 Нативный системный выбор одного или нескольких файлов без ручного пути (`DUD-UI-170`)
- [x] P131 Выбранные файлы остаются черновиком с thumbnail/именем и удалением до общей отправки (`DUD-UI-170`)
- [x] P132 GIF определяется как изображение и получает thumbnail в engine/GUI/TUI (`DUD-FILE-120`)
- [x] P133 После скачивания виден точный путь и доступно «Показать в Finder/файловом менеджере» (`DUD-FILE-140`)
- [x] P134 Вся текстовая лента выделяется и копируется системными средствами (`DUD-UI-171`)
- [x] P135 «ИСКАТЬ» сам выводит приватную подсеть и запускает ограниченный параллельный scan (`DUD-NET-150`)
- [x] P136 AppBar и status strip объединены в один компактный хедер (`DUD-UI-171`)
- [x] P137 Enter вставляет перенос; Cmd+Enter / Ctrl+Enter отправляет (`DUD-UI-171`)
- [x] P138 Attach/send — иконки скрепки и самолётика с tooltip (`DUD-UI-171`)
- [x] P139 Новая единая иконка Дудки для macOS/iOS/Android/Windows (`DUD-UI-172`)

**Фаза 8 готова:** вложения управляются как черновик, desktop-действия нативны, чат не тратит место и текст копируется.

---

## Не делаем (пока явно не попросят)

Облако, Доверие/Диалоги runtime, mDNS-only, голос/звонки, E2E-маркетинг, регистрация, доска в «Делах» как бэклог, callhome.
