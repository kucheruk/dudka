# Как поставить Дудку семье за 5 минут

Без аккаунтов Дудки, без облака, без «зарегистрируйся». Нужен только общий Wi‑Fi квартиры.

## Что получите

На каждом устройстве — клиент. В одном Wi‑Fi люди видят друг друга и шлют текст/файлы.
Ник выбирается локально при первом запуске.

## Быстрый путь (macOS — самый простой сегодня)

1. Соберите или возьмите готовый архив:

```bash
./scripts/build_macos_app.sh
# → dist/dudka-macos.zip
```

2. Передайте `dudka-macos.zip` по AirDrop / флешке на Mac семьи.
3. Распакуйте → откройте `dudka.app` (если Gatekeeper ругается: ПКМ → Открыть).
4. Введите ник → сразу чат. Второй Mac в том же Wi‑Fi — то же самое.
5. Напишите строку → **ОТПРАВИТЬ**. Готово.

Цель: от zip до первого сообщения ≤ 5 минут.

## Linux (терминал)

```bash
./scripts/build_linux_tui.sh
# скопируйте dist/dudkad-linux-* и dist/dudka-linux-* на машину семьи
./dudkad -data-dir ~/.dudka -name Маша &
./dudka -engine 127.0.0.1:17880   # порт смотрите в логе ready / listen=
```

Два Linux-ноутбука в одном Wi‑Fi — два процесса `dudkad`+`dudka`.

## Windows

```bash
./scripts/build_windows_app.sh
# → dist/dudkad-windows-amd64.exe + dist/dudka-windows-amd64.exe
```

Запустите `dudkad-windows-amd64.exe`, затем TUI `dudka-windows-amd64.exe -engine 127.0.0.1:<port>`.
Flutter GUI — соберите на Windows-хосте (см. `dist/BUILD-WINDOWS.md`).

## Android

```bash
./scripts/build_android_apk.sh
adb install -r dist/dudka-android.apk
```

Или передайте APK по USB/файлообмену и откройте на телефоне (разрешить установку из неизвестных источников).
Пока GUI на телефоне удобнее стыковать с `dudkad` на десктопе в той же LAN (см. `dist/BUILD-ANDROID.md`).

## iOS

Unsigned build: `./scripts/build_ios_app.sh`.  
Установка на iPhone семьи — ad-hoc или TestFlight: [`build-ios.md`](build-ios.md) (нужен Apple ID / Team владельца, не аккаунт Дудки).

## Правила квартиры

- Все устройства в **одном Wi‑Fi** (гость-сеть часто режет client isolation — тогда «НИКОГО РЯДОМ»).
- Не нужен интернет; Дудка не ходит в облако.
- Ник можно сменить в настройках / TUI — старые сообщения не переписываются.

## Если не видит соседей

1. Тот же SSID? Не VPN «весь трафик»?
2. На роутере выключить AP/client isolation.
3. Desktop: кнопка **ИСКАТЬ** / `POST /scan` (TUI/API).
4. Смоук по платформам: [`platform-smoke.md`](platform-smoke.md).
