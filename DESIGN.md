---
name: ДУДКА
description: Локальный квартирный чат — rhythm-machine panel + BBS-lite без фанатизма
colors:
  panel: "#1A1A1A"
  panel-deep: "#0E0E0E"
  silkscreen: "#F2F2F2"
  silkscreen-dim: "#8A8A8A"
  step-red: "#FF3B30"
  step-orange: "#FF9A00"
  step-yellow: "#FFD600"
  step-white: "#F2F2F2"
  led-active: "#FF4500"
  led-idle: "#3A3A3A"
  segment: "#FF3B30"
  danger: "#FF3B30"
  ok: "#FFD600"
typography:
  display:
    fontFamily: "JetBrains Mono, UI Monospace, monospace"
    fontSize: "22px"
    fontWeight: 700
    lineHeight: 1.15
    letterSpacing: "0.04em"
  body:
    fontFamily: "JetBrains Mono, UI Monospace, monospace"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.35
    letterSpacing: "0.01em"
  label:
    fontFamily: "JetBrains Mono, UI Monospace, monospace"
    fontSize: "11px"
    fontWeight: 500
    lineHeight: 1.2
    letterSpacing: "0.12em"
  segment:
    fontFamily: "JetBrains Mono, UI Monospace, monospace"
    fontSize: "28px"
    fontWeight: 700
    lineHeight: 1
    letterSpacing: "0.08em"
rounded:
  none: "0px"
  sm: "2px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "24px"
components:
  button-primary:
    backgroundColor: "{colors.led-active}"
    textColor: "{colors.panel-deep}"
    rounded: "{rounded.none}"
    padding: "12px 20px"
  button-ghost:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.silkscreen}"
    rounded: "{rounded.none}"
    padding: "12px 20px"
  input-compose:
    backgroundColor: "{colors.panel-deep}"
    textColor: "{colors.silkscreen}"
    rounded: "{rounded.none}"
    padding: "12px 16px"
---

# DESIGN.md — ДУДКА

Статус: seed (до первой реализации). Мир: early-80s rhythm machine step row × BBS/Terminate-lite.  
Quality bar: `.impeccable/quality-bar/drum-machine-board.png`, `drum-machine-hero.png`.  
Seed: `c3406aa1` · chosen: `signals-instruments-drum-machine-step-row`.

## Overview

ДУДКА выглядит как спокойная charcoal-панель ритм-машины: silkscreen-лейблы, сегментный статус, цветные «шаги» как индикаторы жизни сети — не как игрушка и не как музей CRT. Операционка ленты — BBS/Terminate-lite (плотный mono-feed + список online), без ANSI-арт, scanlines и пиксель-шрифтов 8×8.

Visitor mode на всех GUI-поверхностях: **Operate**. Бренд живёт в материале панели и типографике; навигация и жесты остаются нативными для iOS/Android/desktop.

## Colors

- Фон панели `{colors.panel}` / глубже `{colors.panel-deep}`.
- Текст и рамки — silkscreen `{colors.silkscreen}` / dim `{colors.silkscreen-dim}`.
- Акцент действия и LED «сейчас» — `{colors.led-active}`; сегментные цифры — `{colors.segment}`.
- Step-quarter для прогресса/присутствия: red → orange → yellow → white.
- Никаких фиолетовых SaaS-градиентов, неонового glow-театра и «мессенджерной» зелени.

Dark — единственная схема MVP (панель всегда тёмная). Light Mode не изобретать без отдельного решения.

## Typography

Всё UI — моноширинный стек (JetBrains Mono или системный mono).  
Лейблы — UPPERCASE + letter-spacing (silkscreen).  
Тело ленты — обычный регистр, читаемый; не орать caps на каждое сообщение.  
На iOS/Android уважать Dynamic Type / font scale: масштабировать роли, не ломать mono.

## Layout

- **Phone:** сверху status strip (ДУДКА · онлайн N · Wi‑Fi), затем горизонтальный/сворачиваемый peer strip, основная колонка — лента, снизу compose + attach.
- **Tablet/desktop:** dual-pane Terminate-lite — слева peers (узкая колонка), справа лента+compose.
- **Linux:** основной клиент — выразительная текстовая rhythm-machine топология
  (peers | feed | input), полноценная без GUI.
- **Windows TUI:** дополнительный инструмент диагностики.
- Safe areas / system bars / IME insets обязательны; кастомный tab bar на 5 разделов — anti-goal для MVP (один главный экран).

## Elevation & Depth

Плоская панель. Разделение — 1px silkscreen-линии и tonal panels, не тени и не glassmorphism.  
LED «горит» насыщенностью цвета, не blur-glow (лёгкий акцент допустим только на сегментном readout, без bloom на весь экран).

## Shapes

Радиус ≈ 0–2px. Кнопки и поля — прямоугольные блоки панели.  
Step pads — короткие прямоугольники в ряд (группы по 4 визуально), не pills.

## Components

- **Status strip:** имя продукта, count online (segment-style), короткий статус сети.
- **Peer list:** ник + led idle/active; без аватаров-фото в MVP.
- **Message row:** время · ник · текст; файл — имя, размер, step-progress, thumbnail для image/*.
- **Compose:** многострочное поле + attach-скрепка (ghost) + send-самолётик
  (primary); tooltip сохраняет глагол «ОТПРАВИТЬ». Выбранные файлы видны над
  полем как удаляемый черновик, а не публикуются сразу.
- **Update:** проверенный desktop-пакет включает одну компактную активную кнопку
  `АПДЕЙТ X.Y.Z` в общем хедере рядом с настройками. Пока пакет скачивается или
  не прошёл проверку, кнопки нет; постоянный баннер и модалка не нужны.
- **Desktop lifecycle:** крестик скрывает окно в трей. Трей содержит только
  «Открыть Дудку» и «Выйти»; настройка автозапуска расположена под ником.
- **Unread badge:** системный красный badge с коротким числом непрочитанных на
  Dock/taskbar/launcher; он не становится отдельным виджетом внутри хедера.
- **App icon:** четыре сцепленных угловатых блока — красный, оранжевый,
  off-white и жёлтый — вокруг ломаного чёрного центра на charcoal. Это
  аппаратная метка обмена, не буквальная труба, не speech bubble и не
  контурная иллюстрация.
- **Empty:** «НИКОГО РЯДОМ» + одна secondary «ИСКАТЬ» (subnet scan), без иллюстраций-баннеров.
- **First-run:** один экран ника; дальше сразу чат.

## Do's and Don'ts

**Do**
- Открыл → сразу лента.
- Честно показывать отсутствие peers и отсутствие Wi‑Fi.
- Держать TUI и GUI в одной информационной грамматике.
- Крафт уровня quality-bar: плотность, чёткая сетка, hardware-логика индикаторов.

**Don't**
- Регистрация, облачные копирайты, комнаты/коды, промо-баннеры, сложные настройки.
- CRT/scanline/ANSI-косплей «ради вайба».
- Карточки-мессенджеры, floating FAB-облака, story-кружки.
- Исходящий интернет в UI, кроме точного update-контура DUD-UPD; никаких
  аккаунтов, telemetry и облачных badges.
