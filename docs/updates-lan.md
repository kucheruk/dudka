# Offline updates over LAN (P099)

Без cloud: положить артефакт в `data-dir/updates/` через loopback API, сосед забирает файл
обычным file-announce / копированием.

```bash
# на машине с новой сборкой
curl -s -X POST http://127.0.0.1:17880/updates \
  -d '{"name":"dudka-macos.zip","content_b64":"…"}'
curl -s http://127.0.0.1:17880/updates
# → {"updates":[{"name":"dudka-macos.zip","size":…}]}
```

Контракт: `./scripts/lan_updates_test.sh`.
