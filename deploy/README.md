# Веб-Дудка на сервере Студии

Публичная статика лежит в `/srv/zamoo.team/dudka/web/`. Сигнальный бинарь
лежит в `/opt/dudka-signal/dudka-signal` и слушает только
`127.0.0.1:5251`. Caddy открывает наружу только точный путь
`https://zamoo.team/dudka/signal`.

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

Системный unit: `deploy/dudka-signal.service`. Фрагмент Caddy:
`deploy/Caddyfile.fragment`.
