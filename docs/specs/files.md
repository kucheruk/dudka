# DUD-FILE — Файлы

Статус документа: Draft  
Владелец: ДУДКА  
Последнее смысловое изменение: 2026-07-27

## Назначение и границы

Объявление и передача файлов peer-to-peer в LAN с прогрессом, отменой и превью изображений.

### Non-goals

- Облачное хранение и CDN.
- Антивирусный сканер как продуктовая фича (ОС сама решает).
- Жёсткий лимит размера как отказ по умолчанию.

## Требования

### DUD-FILE-101

Priority: P0  
Status: Draft

Отправитель публикует в чат file-announce: `file_id`, имя, размер байт, mime, content hash, `peer_id` источника. Получатели видят запись в ленте до завершения скачивания.

Проверка:

- announce виден без автоматической полной загрузки бинарника на всех; *(P050: `POST /files/announce` → `type=file_announce` в `GET /messages` у peers; без `GET /files/{id}` body)*
- evidence: protocol test (`scripts/file_announce_test.sh`).

Зависимости: DUD-CHAT-101  
ADR: не требуется

### DUD-FILE-110

Priority: P0  
Status: Partial

Получатель инициирует скачивание чанками у источника (или у peer, подтвердившего наличие полного файла). UI показывает прогресс 0–100% и позволяет отменить; отмена прекращает запись частичного файла или помечает его discarded.

Проверка:

- получатель после fetch имеет полный файл на диске, байты совпадают с источником; *(P051: TCP `file_chunk_req`/`file_chunk`, `POST /files/fetch`, inbox path)*
- во время скачивания API отдаёт прогресс 0–100%; *(P052: `GET /files/transfers`, `wait:false` async fetch; TUI `%` на FILE-строке)*
- отмена на 50% не оставляет «успешный» файл в UX; *(P053: `POST /files/cancel` → `status=cancelled`, partial discarded, не `done`)*
- Flutter: announce/fetch/progress/cancel в shell; *(P067: `EngineClient.announceFile`/`startFetch`/`cancelFetch`, `ChatScreen` СКАЧАТЬ/ОТМЕНА)*
- evidence: `scripts/file_fetch_test.sh`, `scripts/file_progress_test.sh`, `scripts/file_cancel_test.sh`, `./scripts/flutter_files_test.sh` (P067).

Зависимости: DUD-FILE-101  
ADR: не требуется

### DUD-FILE-111

Priority: P0  
Status: Draft

Жёсткого отказа по размеру нет. При размере > 100 MiB UI показывает предупреждение до начала передачи, но не блокирует отправку.

Проверка:

- файл 101 MiB можно начать передавать после warning; *(P054: TUI `WARN>100MiB`, `/fetch` → warning, `/fetch!` или `-fetch` продолжает)*
- evidence: UI test (`scripts/largefile_warn_test.sh`).

Зависимости: DUD-FILE-110  
ADR: не требуется

### DUD-FILE-120

Priority: P0  
Status: Partial

Для mime `image/jpeg`, `image/png`, `image/webp`, `image/heic` (если платформа декодирует) в ленте показывается thumbnail без отдельного обязательного «скачать, чтобы увидеть превью». Полный файл по-прежнему по запросу/автозагрузке политики клиента.

Проверка:

- jpeg/png/webp дают превью в ленте у получателя при живом источнике; *(P056: `thumb_b64` на announce → локальный `thumb_path`, TUI `THUMB <path>`; P068 Flutter `Image.memory` из `thumb_b64`)*
- heic/heif: превью если платформа декодирует (darwin+cgo / ImageIO), иначе честный fallback без фейкового `THUMB`; *(P057 TUI; P068 Flutter метка `HEIC` без фейкового Image)*
- не-image не рисует ложное превью; *(P068)*
- TUI↔TUI: картинка с превью и произвольный бинарник доходят end-to-end; *(P058: `dudka -announce` / `/announce`, `-fetch` / `/fetch`, `scripts/tui_files_e2e_test.sh`)*
- evidence: `scripts/file_thumb_test.sh`, `scripts/file_heic_test.sh`, `scripts/tui_files_e2e_test.sh`, `./scripts/flutter_thumbs_test.sh` (P068).

Зависимости: DUD-FILE-101  
ADR: не требуется

### DUD-FILE-130

Priority: P1  
Status: Draft

После полной загрузки клиент сверяет content hash; mismatch → ошибка «файл повреждён» и файл не считается успешным.

Проверка:

- битый последний чанк / неверный announce hash → ошибка, transfer `error` (не `done`), файл из inbox убран; *(P055: `files.VerifyFile` / `IsCorrupt`, SHA-256 `sha256:<hex>`)*
- evidence: protocol test (`scripts/file_hash_test.sh`).

Зависимости: DUD-FILE-110  
ADR: не требуется
