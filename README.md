# ДУДКА (`dudka`)

Локальный чат квартиры: текст и файлы в одном Wi‑Fi без аккаунтов, облака и исходящего интернета в рантайме.

## Статус

Проектирование / заготовка. Код ещё не начат. Продуктовая правда — [`PRODUCT.md`](PRODUCT.md), визуальный мир — [`DESIGN.md`](DESIGN.md), требования — [`docs/specs/`](docs/specs/).

- Forgejo: <http://winebottle.local:3030/vetinary/dudka>
- Remote: `ssh://git@winebottle.local:2222/vetinary/dudka.git`

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
