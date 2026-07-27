# ДУДКА — ROADMAP

Высокоуровневый план. Детали и ID — в [`docs/specs/`](docs/specs/) и [`docs/design/overview.md`](docs/design/overview.md).  
Приоритеты: **дешевизна разработки → удобство UI → скорость**.

## Фаза 0 — Контур репозитория

- [x] PRODUCT / DESIGN / спеки `DUD-*` / архитектура
- [x] Forgejo-репо `vetinary/dudka`, ветка `master`
- [ ] Минимальный `check.sh` / заготовка CI (по возможности студии)
- [ ] Решение по доске в «Делах» (когда понадобится живая поставка)

**Готово когда:** репо на winebottle, спеки в `master`, агент может клонировать.

## Фаза 1 — Wire + Go engine (без GUI)

Цель: два процесса на одной машине/LAN обмениваются текстом и хвостом.

- Каркас `dudkad`: peer_id, ник, loopback API
- Discovery: UDP broadcast → TCP register → subnet scan (`DUD-NET-*`)
- Текст + tail-keeper на 200 (`DUD-CHAT-*`)
- Контрактные тесты протокола (два loopback-peer)
- Linux TUI v0: peers | лента | ввод (`DUD-UI-150` минимум)

**Готово когда:** TUI↔TUI в одной Wi‑Fi; ноль WAN-connect в тесте (`DUD-NET-101`).

## Фаза 2 — Файлы

- File announce, chunk download, прогресс/отмена, hash (`DUD-FILE-*`)
- Thumbnail для jpeg/png/webp (+ heic где платформа умеет)
- Warning >100 MiB без жёсткого отказа

**Готово когда:** файл и картинка с превью доходят TUI↔TUI (или TUI↔простой второй клиент).

## Фаза 3 — Flutter shell

- Связка Flutter↔engine (loopback/UDS) — выбрать самый дешёвый bind
- Экраны: first-run ник, чат, alone/no_network, мини-настройки (`DUD-UI-*`)
- Визуал по `DESIGN.md` (step-row × BBS-lite, без CRT-фанатизма)
- Desktop layout dual-pane; phone — strip + лента

**Готово когда:** Flutter↔Flutter и Flutter↔TUI в квартирной сети; RU-only UI.

## Фаза 4 — Платформенная упаковка

- Сборки: macOS, Windows, Android; iOS (signing/TestFlight или ad-hoc — по возможности)
- Linux: статический/простой бинарь TUI (+ опционально engine)
- Честный UX при client isolation / нет Wi‑Fi
- Смоук-матрица платформ (`DUD-PRD-102`)

**Готово когда:** семья может поставить и перекинуть строку/файл без аккаунта.

## Фаза 5 — Жёсткость и полировка

- Замеры NFR (`DUD-PRD-120`), подчистка багов discovery на реальных роутерах
- Диагностика: proto mismatch, порт занят, «искать соседей»
- Решение лицензирования (`DUD-PRD-140`) до публичной раздачи бинарников
- Выкинуть временные костыли bind/discovery с exit condition

## Фаза 6 — После MVP (не блокирует v1)

- Комнаты **без кодов** (отдельные каналы на той же LAN)
- Улучшения доставки (ack/retry), если best-effort начнёт бесить
- Store-пакеты / автообновление **offline** (раздача в LAN), без cloud callhome

---

## Явно не в ближайших фазах

Облако, Доверие/Диалоги runtime, mDNS-only, голос/звонки, E2E как продукт, регистрация, callhome в рантайме.
