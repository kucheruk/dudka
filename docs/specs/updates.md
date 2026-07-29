# DUD-UPD — Автообновление приложения

Статус документа: Accepted  
Владелец: ДУДКА  
Последнее смысловое изменение: 2026-07-28

## Назначение и границы

Прямые desktop-сборки macOS и Windows узнают о новой версии с публичного
лендинга, заранее скачивают и проверяют пакет, а затем заменяют приложение по
явному нажатию пользователя. Update-запрос не является telemetry: он не
содержит identity, installation ID, ник, peers или содержимое чата.

### Non-goals

- скрытая принудительная установка;
- обновление iOS в обход App Store/TestFlight;
- установка Android APK без системного подтверждения пользователя;
- updater в `dudkad`, Linux TUI или MCP;
- analytics, rollout targeting и персонализированный manifest.

## Требования

### DUD-UPD-101

Priority: P0  
Status: Accepted

GUI на macOS и Windows проверяет только
`https://zamoo.team/dudka/update.json`. Manifest schema `1` содержит одну
публичную версию и отдельные пакеты `macos-universal` / `windows-amd64`:
HTTPS URL, `sha256`, размер и формат `zip`. Неизвестная schema, не-SemVer
версия, не-HTTPS URL, отсутствующий пакет текущей платформы, неверный размер
или hash отвергаются; размер пакета ограничен 1 GiB.

Проверка:

- unit-тест parsing/validation/version comparison;
- download fixture с верным и неверным SHA-256;
- source gate разрешает только точный manifest URL и запрещает telemetry.

Зависимости: DUD-NET-101  
ADR: не требуется — явное решение владельца 2026-07-28

### DUD-UPD-110

Priority: P0  
Status: Accepted

Проверка запускается после старта GUI и затем не чаще одного раза в 15 минут.
Если manifest новее текущей версии, пакет скачивается в системный cache/temp и
проверяется до показа действия. Кнопка `АПДЕЙТ X.Y.Z` появляется только в
состоянии `ready`, когда локальный архив полностью проверен. Сетевая ошибка
оставляет текущую версию рабочей и не мешает чату, но явно показывается в
desktop-шапке. Полный текст ошибки доступен в подсказке, чтобы его можно было
передать агенту.

Проверка:

- текущая/старая версия не показывает кнопку;
- более новая версия с проверенным пакетом показывает кнопку;
- битый manifest/artifact не показывает кнопку и не меняет приложение;
- ошибка проверки видна, при этом поле ввода и чат остаются рабочими;
- evidence: Flutter unit/widget tests.

Зависимости: DUD-UPD-101  
ADR: не требуется

### DUD-UPD-120

Priority: P0  
Status: Accepted

По нажатию `АПДЕЙТ` updater останавливает bundled `dudkad`, запускает detached
platform helper, закрывает GUI и выдерживает ровно 10 секунд перед заменой.
Helper атомарно сохраняет предыдущую установку, разворачивает проверенный ZIP и
запускает новую версию; при ошибке восстановления старая установка остаётся
запускаемой.

macOS direct-download сборка не использует App Sandbox: замена `.app` рядом с
собой невозможна из sandbox без отдельного privileged installer. Приложение,
запущенное прямо с read-only DMG, просит сначала перенести себя в Applications.
Windows helper работает только с полным GUI ZIP, собранным на Windows host.

Проверка:

- generated macOS/PowerShell helper содержит 10-second delay, wait текущего PID,
  backup/restore и запуск нового executable;
- widget test: нажатие update-кнопки вызывает activation один раз;
- macOS package test: ZIP содержит `.app`, bundled engine универсален;
- ручной smoke N→N+1 на каждой публикуемой desktop-платформе.

Зависимости: DUD-UPD-110  
ADR: не требуется
