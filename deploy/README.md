# Веб-Дудка на сервере Студии

Публичная статика лежит в `/srv/zamoo.team/dudka/web/`. Сигнальный бинарь
лежит в `/opt/dudka-signal/dudka-signal` и слушает только
`127.0.0.1:5251`. Caddy открывает наружу только точный путь
`https://zamoo.team/dudka/signal`. Тот же процесс слушает публичный
`3478/udp` как STUN-only сервис; TURN в нём нет.

Перед обновлением:

1. собрать Linux-бинарь командой `./scripts/build_signal.sh`;
2. проверить `./scripts/web_e2e_test.cjs` с доступным Node.js-пакетом
   `playwright`;
3. проверить новую конфигурацию через `caddy validate` до перезагрузки;
4. запустить `dudka-signal`, получить `ok` с `/health`, затем перезагрузить
   Caddy;
5. открыть две отдельные browser contexts на `/dudka/web/` и доказать:
   до согласия нет WebSocket, после согласия `ОНЛАЙН 2` и текст проходит
   напрямую.

`index.html` не кешируется. Адреса `app.js`, `app.css` и `icon.png` содержат
первые 12 знаков SHA-256. После изменения ассета обновить его `?v=` обязательно;
это проверяет `scripts/web_asset_contract_test.sh`.

Системный unit: `deploy/dudka-signal.service`. Фрагмент Caddy:
`deploy/Caddyfile.fragment`.
