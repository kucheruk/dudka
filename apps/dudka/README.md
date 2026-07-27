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

Cold start: first-run nick (RU) → chat wireframe (status strip + peers + feed). Skip uses hostname / «Прилагательное+Животное». Compose send is P064.

## Checks

```bash
./scripts/flutter_chat_test.sh       # P063
./scripts/flutter_firstrun_test.sh   # P062
./scripts/flutter_skeleton_test.sh   # P061
./scripts/flutter_bind_test.sh       # P060 bind contract
```
