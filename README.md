# ДУДКА (`dudka`)

Локальный чат квартиры: текст и файлы в одном Wi‑Fi без аккаунтов, облака и исходящего интернета в рантайме.

## Статус

**Фаза 1 закрыта:** engine discovery + текст + Linux TUI; `./scripts/check.sh` гоняет unit + multi-peer protocol tests.  
Продуктовая правда — [`PRODUCT.md`](PRODUCT.md), визуальный мир — [`DESIGN.md`](DESIGN.md), требования — [`docs/specs/`](docs/specs/).

- Forgejo: <http://winebottle.local:3030/vetinary/dudka>
- Remote: `ssh://git@winebottle.local:2222/vetinary/dudka.git`
- Go module: `dudka` (`go.mod`)

## Сборка stub

```bash
go build -o dist/dudkad ./cmd/dudkad   # engine
go build -o dist/dudka ./cmd/dudka     # Linux TUI
./dist/dudkad -data-dir /tmp/dudka-demo -name Вася -listen 127.0.0.1:17880
# → ready peer_id=<uuid> name=Вася
# → UDP announce :41777 + TCP register; curl /peers → соседи
# curl -s http://127.0.0.1:17880/me → {"peer_id":"…","name":"Вася"}
# curl -s http://127.0.0.1:17880/peers → {"peers":[{"peer_id":"…","updated":false,…}]}
# curl -s http://127.0.0.1:17880/status → proto_major + incompatible[]
# curl -s -X POST http://127.0.0.1:17880/scan -d '{"hosts":["192.168.1.10"],"port":41777}'
# (scan — fallback, когда UDP broadcast отфильтрован)
# публичный seed IP не уходит в WAN: лог wan_refuse (DUD-NET-101)
# ./dist/dudkad -dial-hosts 8.8.8.8 …
# curl -s -X POST http://127.0.0.1:17880/send -d '{"text":"привет"}' → accepted|queued (не «доставлено»)
# curl -s -X POST http://127.0.0.1:17880/files/announce \
#   -d '{"name":"a.txt","mime":"text/plain","hash":"sha256:…","content_b64":"…"}'
#   → file_announce в ленте; blob остаётся у источника (P050/P051)
# curl -s -X POST http://127.0.0.1:17880/files/fetch -d '{"file_id":"…"}'
#   → чанки у источника → полный файл в data-dir/inbox (P051)
# curl -s -X POST …/files/fetch -d '{"file_id":"…","wait":false}' + GET /files/transfers
#   → percent 0–100 во время скачивания (P052)
# ./dist/dudka -engine … -fetch <file_id>   # кадры с NN% до 100%
# curl -s http://127.0.0.1:17880/messages → лента у всех online
# текст > 4000 code points → 4xx + понятная ошибка (P031)
# curl -s http://127.0.0.1:17880/tail → хвост ≤200 + keeper_id (после join синхронизируется с keeper)
# уход keeper → peer_gone / перевыбор; новый peer всё ещё получает хвост (P034)
./dist/dudka -engine 127.0.0.1:17880
# → dudka <ver> + status + peers + FEED + INPUT
# ./dist/dudka -engine 127.0.0.1:17880 -send "привет"   # одна отправка
# ./dist/dudka -engine 127.0.0.1:17880 -nick "Вася"     # смена ника
# ./dist/dudka -watch -engine 127.0.0.1:17880           # Enter = send · /nick Имя
```

Каркас: `cmd/dudkad`, `cmd/dudka`, `internal/{version,identity,loopback,discovery,chat,tui}`.

## Локальный гейт

Единая проверка репозитория (локально и в CI):

```bash
./scripts/check.sh
```

Гейт запускает `go test ./...`, затем multi-peer **protocol tests** (`./scripts/protocol_tests.sh`: announce/peers/send/tail/WAN/TUI exchange и др., 2+ peer).  
Мета-контракт P045: `./scripts/protocol_gate_test.sh`.  
Прочие контракты по задачам: `./scripts/check_test.sh`, `./scripts/gomod_test.sh`, `./scripts/skeleton_test.sh`, `./scripts/peerid_test.sh`, `./scripts/displayname_test.sh`, `./scripts/health_test.sh`, `./scripts/me_test.sh`, `./scripts/nick_test.sh`, `./scripts/send_length_test.sh`, `./scripts/file_announce_test.sh`, `./scripts/file_fetch_test.sh`, `./scripts/file_progress_test.sh`, `./scripts/tui_peers_test.sh`, `./scripts/tui_feed_test.sh`, `./scripts/tui_nick_test.sh`.

## Зачем

Семья айтишников устала от блокировок и сложности обычных мессенджеров, когда нужно просто перекинуть строку или файл через комнату. ДУДКА работает только в LAN: открыл приложение — уже в общем чате с теми, кто онлайн рядом.

## Платформы

| Платформа | UI |
| --- | --- |
| iOS, Android, Windows, macOS | Flutter (грамматика из `DESIGN.md`) |
| Linux | текстовый TUI (Go) |

## Стек (зафиксирован на проектировании)

- **Go** — протокол, discovery, хвост истории, файлы, Linux TUI, встроенный engine для GUI.
- **Flutter** — GUI shell; говорит с локальным Go-engine только через loopback.
- Discovery: UDP broadcast + TCP register + subnet scan; **не** mDNS.
- Рантайм без интернета (регуляторный контур, не деталь).

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

Открытое решение: см. [`docs/specs/product.md`](docs/specs/product.md) (DUD-PRD-140). Пока продукт намеренно offline — callhome в рантайме запрещён требованиями сети.
