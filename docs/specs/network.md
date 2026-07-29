# DUD-NET — Сеть и discovery

Статус документа: Draft  
Владелец: ДУДКА  
Последнее смысловое изменение: 2026-07-27

## Назначение и границы

Единый WebRTC-контур браузеров и приложений. Не описывает UX-копирайт
(см. DUD-UI) и семантику сообщений (DUD-CHAT).

### Non-goals

- mDNS / DNS-SD / Bonjour как механизм discovery.
- Relays, NAT traversal через интернет, overlay (Tailscale и т.п.).
- Работа через AP client isolation (честный отказ).

## Требования

### DUD-NET-101

Priority: P0
Status: Replaced by DUD-NET-170

Engine в рантайме не устанавливает исходящие соединения к адресам вне
link-local, loopback и частных диапазонов IPv4/IPv6 (RFC1918, Unique Local).
Flutter GUI имеет одно явное исключение: desktop updater читает HTTPS manifest
`https://zamoo.team/dudka/update.json` и затем скачивает указанный им release
artifact по DUD-UPD-101. Попытка конфигурации другого «облачного» endpoint
игнорируется или отвергается.

Проверка:

- интеграционный тест: блокировщик/прокси фиксирует ноль WAN-попыток engine и
  ноль GUI-запросов, кроме разрешённого update manifest/artifact;
- негатив: подсовывание публичного IP в конфиг не приводит к connect; *(P025: `DialHosts` / `CheckDialHost`, лог `wan_refuse`)*
- evidence: тест + лог.

Зависимости: DUD-UPD-101 для единственного исключения GUI
ADR: не требуется

### DUD-NET-160

Priority: P0
Status: Accepted

Браузерная Дудка после явного согласия пользователя имеет два дополнительных
WAN-исключения: `wss://zamoo.team/dudka/signal` и
`stun:zamoo.team:3478` по UDP. WebSocket принимает только WebRTC
offer/answer/ICE и случайные ID вкладок. STUN возвращает только наблюдаемый
IP и UDP-порт. Сообщения, файлы, имена и история через них запрещены. TURN
не используется.

Проверка:

- до согласия соединения нет;
- allowlist содержит точный same-origin WebSocket path и один STUN Студии;
- WebRTC config не содержит сторонних STUN или TURN;
- signaling service отвергает прикладные типы данных;
- STUN отвечает только на RFC 5389 Binding Request;
- evidence: DUD-WEB-101/110/140 tests.

Зависимости: DUD-WEB-101, DUD-WEB-140
ADR: `docs/adr/0001-browser-webrtc-signaling.md`

### DUD-NET-110

Priority: P0  
Status: Replaced by DUD-NET-170

Discovery использует стек: (1) UDP broadcast announce, (2) TCP register в ответ/инициативно, (3) subnet scan как fallback или по действию пользователя. mDNS не используется.

Проверка:

- два peer находят друг друга при выключенном mDNS-стеке ОС;
- при глушении broadcast остаётся путь через scan;
- evidence: автотест + ручной прогон в квартирной сети.

Зависимости: нет  
ADR: не требуется

### DUD-NET-111

Priority: P0  
Status: Replaced by DUD-NET-170

Announce и register содержат: `peer_id`, `display_name`, `proto_major`, `proto_minor`, TCP port для сессии, `instance_id` (меняется при рестарте процесса).

Проверка:

- схема зафиксирована в `docs/specs/` или generated contract;
- несовместимый `proto_major` не приводит к молчаливой порче состояния (UI показывает несовместимость);
- evidence: contract test.

Зависимости: DUD-NET-110  
ADR: не требуется

### DUD-NET-120

Priority: P0  
Status: Replaced by DUD-NET-170

По умолчанию LAN bind использует UDP/TCP порт `41777`. Если порт занят, engine выбирает свободный порт из документированного диапазона и объявляет его в announce; UI показывает фактический порт в диагностике.

Проверка:

- конфликт порта не роняет процесс без диагностики;
- второй инстанс на той же машине либо соединяется осмысленно, либо явно сообщает о конфликте;
- evidence: тест занятого порта.

Зависимости: DUD-NET-110  
ADR: не требуется

### DUD-NET-130

Priority: P0  
Status: Draft

Loopback API engine↔GUI принимает соединения только с `127.0.0.1` / `::1` или UDS с правами пользователя; LAN-клиенты к loopback API не допускаются.

Проверка:

- connect с другого хоста к loopback-порту отвергается;
- evidence: сетевой тест.

Зависимости: нет  
ADR: не требуется

### DUD-NET-140

Priority: P1  
Status: Draft

При отсутствии Wi‑Fi/LAN интерфейса UI получает состояние `no_network` ≤ 1 s после детекта; при peers=0 и живой LAN — `alone`.

Проверка:

- airplane mode → `no_network`;
- Wi‑Fi есть, peers нет → `alone`, не `no_network`;
- evidence: unit/integration на статусах.

Зависимости: DUD-UI-120  
ADR: не требуется

### DUD-NET-150

Priority: P0
Status: Replaced by DUD-NET-170

Пользовательская команда «ИСКАТЬ» не требует ручной передачи hosts/CIDR.
При пустом `POST /scan` engine выбирает наиболее вероятный активный RFC1918
интерфейс, ограничивает диапазон максимум одним `/24` вокруг локального адреса
и пробует адреса параллельно в пределах общего timeout. Явные `hosts`/`cidr`
для диагностики сохраняются.

Проверка:

- пустой request не отвечает `scan requires hosts or private cidr`;
- network/broadcast/self адреса не пробуются;
- public CIDR по-прежнему отвергается;
- evidence: `internal/discovery/scan_test.go`, `scripts/scan_test.sh`.

Зависимости: DUD-NET-101, DUD-NET-110
ADR: не требуется

### DUD-NET-170

Priority: P0
Status: Accepted

Все пользовательские клиенты Дудки — браузер, Flutter desktop и Linux TUI —
входят в один полносвязный WebRTC mesh через
`wss://zamoo.team/dudka/signal` и `stun:zamoo.team:3478`.

Проверка:

- до явного согласия нативный engine не открывает signaling и STUN;
- браузер и приложение видят друг друга в списке участников;
- текст и файл проходят в обе стороны через совместимые DataChannel-пакеты;
- два приложения обмениваются текстом без UDP/TCP `41777`;
- signaling отвергает прикладные данные, TURN отсутствует;
- «ИСКАТЬ» переподключает signaling и не сканирует подсеть;
- evidence: `internal/rtcmesh/client_test.go`,
  `scripts/native_web_e2e_test.cjs`.

Зависимости: DUD-WEB-110, DUD-WEB-120, DUD-WEB-130
ADR: `docs/adr/0002-one-webrtc-mesh.md`
