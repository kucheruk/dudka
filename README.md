# ДУДКА (`dudka`)

Локальный чат квартиры: текст и файлы в одном Wi‑Fi без аккаунтов, облака и исходящего интернета в рантайме.

## Статус

Фаза 0 (контур): модуль и stub-бинарники на месте. Продуктовая правда — [`PRODUCT.md`](PRODUCT.md), визуальный мир — [`DESIGN.md`](DESIGN.md), требования — [`docs/specs/`](docs/specs/).

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
./dist/dudka    # → dudka 0.0.0-dev
```

Каркас: `cmd/dudkad`, `cmd/dudka`, `internal/{version,identity,loopback,discovery}`.

## Локальный гейт

Единая проверка репозитория (локально и в CI):

```bash
./scripts/check.sh
```

Гейт запускает `go test ./...`. Контракты: `./scripts/check_test.sh`, `./scripts/gomod_test.sh`, `./scripts/skeleton_test.sh`, `./scripts/peerid_test.sh`, `./scripts/displayname_test.sh`, `./scripts/health_test.sh`, `./scripts/me_test.sh`, `./scripts/nick_test.sh`, `./scripts/announce_test.sh`, `./scripts/peers_test.sh`, `./scripts/instance_test.sh`, `./scripts/proto_test.sh`, `./scripts/scan_test.sh`.

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
