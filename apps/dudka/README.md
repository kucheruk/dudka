# apps/dudka — Flutter shell (spike)

Thin UI over `dudkad` loopback. Bind decision: [`docs/design/flutter-bind.md`](../../docs/design/flutter-bind.md) (P060).

## Hello `/me`

```bash
# from repo root
./scripts/run_flutter_spike.sh
# or tests:
./scripts/flutter_bind_test.sh
```

Pass engine URL: `--dart-define=DUDKA_ENGINE=http://127.0.0.1:PORT`.
