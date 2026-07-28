# Смоук-таблица платформ (P086)

Обновлять при каждой поставке артефактов фазы 4.  
Легенда: ✅ проверено руками/скриптом · ⚠️ частично · ❌ не проверено · 🚫 N/A

| Платформа | Артефакт | Сборка | Установка | Текст 2 peer | Файл 2 peer | Дата / SHA | Кто |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Linux TUI | `dudka-linux-amd64.tar.gz` + `install.sh` | ✅ `build_linux_tui_test.sh` | ✅ user install, no apt | ✅ protocol_tests | ✅ protocol/files | 2026-07-28 / v0.5.2 | agent |
| macOS GUI | `dudka.app` / `dudka-macos-universal.zip` / DMG | ✅ `build_macos_app_test.sh` | ✅ open .app | ✅ flutter_ff / LAN | ✅ flutter_ff | 2026-07-28 | agent |
| Windows GUI | portable `dudka-windows-amd64.zip` | ✅ native CI build | ✅ распаковка + запуск GUI | ✅ Flutter tests | ⚠️ live LAN | 2026-07-28 / P156 release CI | agent |
| Android | `dudka-android.apk` | ✅ `build_android_apk_test.sh` | ⚠️ sideload APK; engine embed TBD | ❌ на телефоне | ❌ | 2026-07-28 / P083 | agent |
| iOS | `dudka-ios-Runner.app` unsigned | ✅ `build_ios_app_test.sh` | 🚫 нет codesign/device на CI Mac | ❌ | ❌ | 2026-07-28 / P084 | agent |

## Как обновлять

После `./scripts/build_*` и ручного/скриптового прогона — новая строка или правка ячейки + SHA `git rev-parse --short HEAD`.

## Известные дыры

- iOS physical install: нужен Apple Team (см. `docs/build-ios.md`).
- Android/iOS: subprocess `dudkad` sidecar ещё не упакован в mobile bundle.
- Windows GUI и Linux TUI проходят нативную сборку в GitHub Actions; живой
  двухмашинный smoke после установки всё ещё нужен.
- Auto-update: macOS package/helper covered by build + unit tests; live
  N→N+1 smoke needs the next published version.

## P093 home router

| Проверка | Результат |
| --- | --- |
| 2× dudkad same Wi‑Fi SSID (lab: SO_REUSEPORT :41779) | ✅ peers + text (nfr/protocol) |
| Client isolation / guest Wi‑Fi | не гонялось на agent host — семья: выключить isolation |
| Баги | нет блокирующих; при isolation — «НИКОГО РЯДОМ» ожидаемо |
