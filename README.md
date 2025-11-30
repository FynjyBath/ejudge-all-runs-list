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

Выводится JSON-массив строк с указанием контеста, пользователя, задачи, статуса и баллов. Строки отсортированы по времени отправки (самые новые сверху). Например:

```json
[
  {
    "contest": "Algorithms 101",
    "contest_id": 42,
    "run_id": 123,
    "submitted_at": "2024-05-28T10:00:12Z",
    "user": "ivan",
    "problem": "A",
    "result": "OK 100",
    "contest_url": "https://example.com/new-judge?contest_id=42"
  },
  {
    "contest": "Algorithms 101",
    "contest_id": 42,
    "run_id": 122,
    "submitted_at": "2024-05-28T09:55:00Z",
    "user": "maria",
    "problem": "B",
    "result": "WA 0",
    "contest_url": "https://example.com/new-judge?contest_id=42"
  }
]
```
