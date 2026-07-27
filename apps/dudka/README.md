# apps/dudka — Flutter shell

macOS-first skeleton over `dudkad` loopback ([`docs/design/flutter-bind.md`](../../docs/design/flutter-bind.md)).

## Run

```bash
# from repo root — builds dudkad, opens macOS app with spawned engine
./scripts/run_flutter_spike.sh

# or attach to an already-running engine:
cd apps/dudka
flutter run -d macos --dart-define=DUDKA_ENGINE=http://127.0.0.1:17880

# or spawn engine from the app:
flutter run -d macos --dart-define=DUDKAD_BIN=$PWD/../../dist/dudkad
```

Cold start: first-run nick (RU) → chat (status/peers/feed) → compose «ДУНУТЬ» (`POST /send`). Skip uses hostname / «Прилагательное+Животное».

Layout: wide (≥700 dp) dual-pane peers|feed; narrow — horizontal peer strip. Resize keeps compose draft.

## Checks

```bash
./scripts/ru_ui_test.sh              # P072 Russian user-facing GUI/TUI strings
./scripts/flutter_ff_test.sh         # P071 Flutter↔Flutter text+file (two peers)
./scripts/flutter_layout_test.sh     # P070 wide dual-pane / narrow peer strip
./scripts/flutter_theme_test.sh      # P069 DESIGN.md theme + step-progress
./scripts/flutter_thumbs_test.sh     # P068 image thumbs in feed
./scripts/flutter_files_test.sh      # P067 file announce/fetch/cancel
./scripts/flutter_settings_test.sh   # P066 mini-settings nick
./scripts/flutter_seek_test.sh       # P065 alone/no_network + ИСКАТЬ
./scripts/flutter_send_test.sh       # P064 Flutter↔TUI text
./scripts/flutter_chat_test.sh       # P063
./scripts/flutter_firstrun_test.sh   # P062
./scripts/flutter_skeleton_test.sh   # P061
./scripts/flutter_bind_test.sh       # P060 bind contract
```
