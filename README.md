# ejudge-all-runs-list

Утилита на Go, которая собирает список посылок по всем указанным контестам Ejudge через REST API `/ej/api/v1/master/list-runs-json`.

## Требования

* Go 1.23+
* Токен с правами доступа к master API (`Authorization` заголовок).
* Базовый URL вашего сервера Ejudge (например, `https://example.com`).

## Запуск

```json
{
  "base_url": "https://example.com",
  "token": "Bearer <ваш_токен>",
  "contests": "42,43",
  "contest_file": "contests.txt",
  "contest_dir": "/home/judges",
  "page_size": 500,
  "field_mask": 0
}
```

Запуск с конфигурационным файлом и фильтром:

```bash
go run ./main.go --config config.json --filter "user_login like 'ivan%'"
```

В конфигурации можно указать один или несколько источников идентификаторов контестов: `contests`, `contest_file` или `contest_dir`.

Флаги:

* `--config` — путь к JSON-конфигурации (по умолчанию `config.json`).
* `--filter` — `filter_expr`, который передаётся в Ejudge для выборки посылок.

Выводится таблица по каждому контесту: `run_id`, пользователь, задача, статус и баллы. Строки отсортированы по времени отправки (самые новые сверху). Например:

```
Contest 42 — Algorithms 101 (runs: 3)
run_id    user   problem   status   score
123       ivan   A         OK       100
122       maria  B         WA       0
121       guest  A         CE       0
```
