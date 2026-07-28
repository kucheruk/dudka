# Смоук-таблица платформ (P086)

Обновлять при каждой поставке артефактов фазы 4.  
Легенда: ✅ проверено руками/скриптом · ⚠️ частично · ❌ не проверено · 🚫 N/A

| Платформа | Артефакт | Сборка | Установка | Текст 2 peer | Файл 2 peer | Дата / SHA | Кто |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Linux TUI | `dudka-linux-*` + `dudkad-linux-*` | ✅ `build_linux_tui_test.sh` | ✅ copy+run | ✅ protocol_tests | ✅ protocol/files | 2026-07-28 / см. master | agent |
| macOS GUI | `dudka.app` / `dudka-macos-universal.zip` / DMG | ✅ `build_macos_app_test.sh` | ✅ open .app | ✅ flutter_ff / LAN | ✅ flutter_ff | 2026-07-28 | agent |
| Windows | `dudkad/dudka-windows-*.exe`; GUI ZIP on Windows host | ✅ PE cross-build | ⚠️ GUI только на Win-хосте | ⚠️ engine PE; GUI TBD on Win | ⚠️ | 2026-07-28 / P082 | agent |
| Android | `dudka-android.apk` | ✅ `build_android_apk_test.sh` | ⚠️ sideload APK; engine embed TBD | ❌ на телефоне | ❌ | 2026-07-28 / P083 | agent |
| iOS | `dudka-ios-Runner.app` unsigned | ✅ `build_ios_app_test.sh` | 🚫 нет codesign/device на CI Mac | ❌ | ❌ | 2026-07-28 / P084 | agent |

## Как обновлять

После `./scripts/build_*` и ручного/скриптового прогона — новая строка или правка ячейки + SHA `git rev-parse --short HEAD`.

## Известные дыры

- iOS physical install: нужен Apple Team (см. `docs/build-ios.md`).
- Android/iOS: subprocess `dudkad` sidecar ещё не упакован в mobile bundle.
- Windows Flutter GUI: собирать на Windows.
- Auto-update: macOS package/helper covered by build + unit tests; live
  N→N+1 smoke needs the next published version. Windows update package remains
  absent from manifest until a full GUI ZIP is built and smoked on Windows.

## P093 home router

| Проверка | Результат |
| --- | --- |
| 2× dudkad same Wi‑Fi SSID (lab: SO_REUSEPORT :41779) | ✅ peers + text (nfr/protocol) |
| Client isolation / guest Wi‑Fi | не гонялось на agent host — семья: выключить isolation |
| Баги | нет блокирующих; при isolation — «НИКОГО РЯДОМ» ожидаемо |
