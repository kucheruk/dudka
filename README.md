# ДУДКА (`dudka`)

Прямой чат квартиры: текст и файлы без аккаунтов и облачного хранения.
После явного согласия signaling и STUN на `zamoo.team` только знакомят
устройства; сообщения и файлы идут напрямую по WebRTC.

## Статус

**Desktop GUI работает на macOS и Windows; Linux получает основной TUI.**
`./scripts/check.sh` гоняет unit +
multi-peer protocol tests.
Продуктовая правда — [`PRODUCT.md`](PRODUCT.md), визуальный мир — [`DESIGN.md`](DESIGN.md), требования — [`docs/specs/`](docs/specs/).

- Forgejo: <http://winebottle.local:3030/vetinary/dudka>
- Remote: `ssh://git@winebottle.local:2222/vetinary/dudka.git`
- GitHub: <https://github.com/kucheruk/dudka> — публичное зеркало; Forgejo остаётся каноническим репозиторием
- Go module: `dudka` (`go.mod`)
- История версий и источник release notes: [`CHANGELOG.md`](CHANGELOG.md)

## Сборка stub

```bash
go build -o dist/dudkad ./cmd/dudkad   # engine
go build -o dist/dudka ./cmd/dudka     # Linux TUI
./dist/dudkad -data-dir /tmp/dudka-demo -name Вася -listen 127.0.0.1:17880
# → ready peer_id=<uuid> name=Вася
# → до явного согласия нет signaling и STUN
# curl -s -X POST http://127.0.0.1:17880/internet-consent
# → WebRTC signaling и STUN запущены; curl /peers → соседи
# curl -s http://127.0.0.1:17880/me → {"peer_id":"…","name":"Вася"}
# curl -s http://127.0.0.1:17880/peers → {"peers":[{"peer_id":"…","updated":false,…}]}
# curl -s http://127.0.0.1:17880/status → proto_major + incompatible[]
# curl -s -X POST http://127.0.0.1:17880/scan -d '{}'
# → повторное знакомство через signaling, без сканирования подсети
# curl -s -X POST http://127.0.0.1:17880/send -d '{"text":"привет"}' → accepted|queued (не «доставлено»)
# curl -s -X POST http://127.0.0.1:17880/files/announce \
#   -d '{"name":"a.txt","mime":"text/plain","hash":"sha256:…","content_b64":"…"}'
#   → file_announce в ленте; blob остаётся у источника (P050/P051)
# curl -s -X POST http://127.0.0.1:17880/files/fetch -d '{"file_id":"…"}'
#   → чанки у источника → полный файл в data-dir/inbox (P051)
# curl -s -X POST …/files/fetch -d '{"file_id":"…","wait":false}' + GET /files/transfers
#   → percent 0–100 во время скачивания (P052)
# curl -s -X POST …/files/cancel -d '{"file_id":"…"}' → cancelled, partial discarded (P053)
# ./dist/dudka -engine … -fetch <file_id>   # кадры с NN% до 100%
# файл >100 MiB: WARN в ленте; /fetch просит /fetch!; -fetch пишет warning и всё равно стартует (P054)
# после полной загрузки сверяется hash; mismatch → «файл повреждён», не success (P055)
# image/jpeg|png|webp: thumb_b64 в announce, файл в data-dir/thumbs, TUI «THUMB <path>» (P056)
# image/heic|heif: превью на darwin+cgo (ImageIO); иначе TUI «HEIC» без фейкового THUMB (P057)
# ./dist/dudka -announce ./pic.jpg | -announce ./a.bin  → лента + thumb для image (P058)
# ./dist/dudka -fetch <file_id>  /  /announce <path> · /fetch <id> в compose (P058 e2e)
# /cancel <file_id> в TUI compose
# curl -s http://127.0.0.1:17880/messages → лента у всех online
# текст > 4000 code points → 4xx + понятная ошибка (P031)
# curl -s http://127.0.0.1:17880/tail → хвост ≤200 + keeper_id (после join синхронизируется с keeper)
# уход keeper → peer_gone / перевыбор; новый peer всё ещё получает хвост (P034)
# Нормальный TUI (по умолчанию в Terminal):
./dist/dudka -engine 127.0.0.1:17880
# или: DUDKA_ENGINE=127.0.0.1:51315 go run ./cmd/dudka
# → alt-screen: status | соседи | лента | compose (Enter = отправить)
# ./dist/dudka -engine … -once          # один plain-кадр (скрипты)
# ./dist/dudka -engine … -send "привет" # one-shot
# ./dist/dudka -engine … -nick "Вася"
# /search                              # найти соседей; /scan и /поиск — алиасы
# ./dist/dudka -watch -engine …         # legacy line-mode для тестов
```

Каркас: `cmd/dudkad`, `cmd/dudka`, `internal/{version,identity,loopback,discovery,chat,tui}`, `apps/dudka` (Flutter shell).

Flutter↔engine (P060–P072): subprocess + HTTP loopback, **macOS-first** shell в `apps/dudka` — DESIGN.md charcoal UI + adaptive dual-pane/peer strip + чат/файлы/превью; Flutter↔Flutter text+file (`./scripts/flutter_ff_test.sh`); RU UI (`./scripts/ru_ui_test.sh`); bind ADR [`docs/design/flutter-bind.md`](docs/design/flutter-bind.md); `./scripts/flutter_*_test.sh`, `./scripts/run_flutter_spike.sh`.

## Linux TUI (P157)

Готовая терминальная версия устанавливается одной строкой:

```bash
curl -fsSL https://zamoo.team/dudka/install.sh | sh
```

Скрипт скачивает точный архив, сверяет SHA-256 и кладёт `dudka` с `dudkad`
в `~/.local/bin`. Пакеты и GUI-зависимости не устанавливаются, `sudo` и
настройка firewall не нужны.

Одна команда → артефакты в `dist/` (cross-compile, `CGO_ENABLED=0`):

```bash
./scripts/build_linux_tui.sh
# → dist/dudka-linux-amd64  (TUI)
# → dist/dudkad-linux-amd64 (engine)
# → dist/dudka-linux-amd64.tar.gz
# → dist/install.sh
# → dist/dudka, dist/dudkad (symlink на текущий GOARCH)
# GOARCH=arm64 ./scripts/build_linux_tui.sh
```

После установки достаточно команды `dudka`: локальный engine запускается
автоматически и хранит имя, историю и файлы в пользовательском каталоге.

## Сборка macOS desktop (P081)

Одна команда → `dist/dudka.app` + update ZIP + DMG (engine `dudkad` внутри):

```bash
./scripts/build_macos_app.sh
open dist/dudka.app
# автоапдейт: dist/dudka-macos-universal.zip
# ручная установка: dist/dudka-macos-universal.dmg
```

Контракт: `./scripts/build_macos_app_test.sh`.

Иконка всех GUI-платформ генерируется из утверждённого
`apps/dudka/assets/branding/app_icon_source.png` одной командой:

```bash
./scripts/generate_app_icons.sh
```

## Автообновление desktop

macOS и полный Windows GUI проверяют только
`https://zamoo.team/dudka/update.json` при старте и раз в 15 минут. Новая
версия заранее скачивается и проверяется по размеру и SHA-256; только после
этого в хедере появляется `АПДЕЙТ X.Y.Z`. По нажатию приложение закрывается,
через 10 секунд проверенный пакет заменяет текущую установку и новая версия
запускается. Manifest: [`docs/update-manifest.example.json`](docs/update-manifest.example.json),
контракт: [`docs/specs/updates.md`](docs/specs/updates.md).

На macOS приложение нужно перенести из DMG в `/Applications` или другую
доступную для записи папку. iOS обновляется через App Store/TestFlight,
Android — системной установкой APK; самодельная фоновая замена там не
используется.

## Выпуск версии

Каждый законченный батч получает новую SemVer-версию и в том же проходе
публикуется на лендинге. Полный обязательный порядок, правила bump и readback —
в разделе «Версии, release notes и деплой» файла [`AGENTS.md`](AGENTS.md).

## Сборка Windows (P082)

```bash
./scripts/build_windows_app.sh
# → dist/dudka-windows-amd64.zip
```

Скрипт выполняется на Windows; в репозитории есть desktop-build workflow.
Пользователь распаковывает portable ZIP и запускает один `dudka.exe` без
терминала и установки. Встроенный engine лежит в служебной подпапке и не
показывается как отдельная программа. Контракт:
`./scripts/build_windows_app_test.sh`.

## Сборка Android (P083)

```bash
./scripts/build_android_apk.sh
# → dist/dudka-android.apk + dist/dudka-android.aab
# adb install -r dist/dudka-android.apk
```

Контракт: `./scripts/build_android_apk_test.sh`. Sidecar-движок на телефоне — см. `dist/BUILD-ANDROID.md`.

## Сборка iOS (P084)

```bash
./scripts/build_ios_app.sh
# → dist/dudka-ios-Runner.app + dist/dudka-ios-unsigned.zip (unsigned device build)
```

Ad-hoc / TestFlight: [`docs/build-ios.md`](docs/build-ios.md). Контракт: `./scripts/build_ios_app_test.sh`.

## Поставить семье за 5 минут (P085)

Короткий гайд без аккаунтов: [`docs/family-install.md`](docs/family-install.md).

Смоук платформ (P086): [`docs/platform-smoke.md`](docs/platform-smoke.md).

## Локальный гейт

Единая проверка репозитория (локально и в CI):

```bash
./scripts/check.sh
```

Гейт запускает `go test ./...`, затем multi-peer **protocol tests** (`./scripts/protocol_tests.sh`: announce/peers/send/tail/WAN/TUI exchange и др., 2+ peer).  
Мета-контракт P045: `./scripts/protocol_gate_test.sh`.  
Interactive TUI (P046): `./scripts/tui_interactive_test.sh`.  
Linux TUI/engine pack (P080): `./scripts/build_linux_tui.sh`, контракт `./scripts/build_linux_tui_test.sh`.  
Прочие контракты по задачам: `./scripts/check_test.sh`, `./scripts/gomod_test.sh`, `./scripts/skeleton_test.sh`, `./scripts/peerid_test.sh`, `./scripts/displayname_test.sh`, `./scripts/health_test.sh`, `./scripts/me_test.sh`, `./scripts/nick_test.sh`, `./scripts/send_length_test.sh`, `./scripts/file_announce_test.sh`, `./scripts/file_fetch_test.sh`, `./scripts/file_progress_test.sh`, `./scripts/file_cancel_test.sh`, `./scripts/largefile_warn_test.sh`, `./scripts/file_thumb_test.sh`, `./scripts/file_heic_test.sh`, `./scripts/tui_files_e2e_test.sh`, `./scripts/tui_peers_test.sh`, `./scripts/tui_feed_test.sh`, `./scripts/tui_nick_test.sh`.

## Зачем

Семья айтишников устала от блокировок и сложности обычных мессенджеров, когда
нужно просто перекинуть строку или файл через комнату. ДУДКА знакомит устройства
через Студию, а текст и файлы передаёт напрямую по WebRTC.

## Платформы

| Платформа | UI |
| --- | --- |
| iOS, Android, Windows, macOS, Linux | Flutter (грамматика из `DESIGN.md`) |
| Linux/Windows TUI | дополнительный терминальный инструмент (Go) |

## Стек (зафиксирован на проектировании)

- **Go** — протокол, discovery, хвост истории, файлы, дополнительный TUI,
  встроенный engine для GUI.
- **Flutter** — GUI shell; говорит с локальным Go-engine только через loopback.
- Discovery: узкий signaling Студии; транспорт: WebRTC DataChannel.
- Signaling и STUN запускаются только после явного согласия; TURN отсутствует.

## Документы

| Файл | Содержание |
| --- | --- |
| [`PRODUCT.md`](PRODUCT.md) | пользователи, границы, принципы |
| [`DESIGN.md`](DESIGN.md) | визуальная система |
| [`AGENTS.md`](AGENTS.md) | конституция репо для агентов |
| [`docs/design/overview.md`](docs/design/overview.md) | архитектура MVP |
| [`docs/specs/`](docs/specs/) | адресуемые требования `DUD-*` |
| [`ROADMAP.md`](ROADMAP.md) | **единственный бэклог** — чеклист `P001`…`P100` (без доски в Делах) |

## Non-goals MVP

Комнаты, коды, E2E как продукт, голос/звонки, облако, Доверие/Диалоги как runtime-зависимости, callhome.

## Лицензирование

Исходники опубликованы для проверки, личного изучения и некоммерческого
использования по **PolyForm Noncommercial 1.0.0**. Это source-available, не
OSI Open Source: коммерческое использование требует отдельного разрешения.
См. [`LICENSE.md`](LICENSE.md) и [`docs/licensing.md`](docs/licensing.md).
Рантайм остаётся Community без callhome по
[`DUD-PRD-140`](docs/specs/product.md#dud-prd-140).
