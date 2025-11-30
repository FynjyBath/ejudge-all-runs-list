# ejudge-all-runs-list

Утилита на Go, которая собирает список посылок по всем указанным контестам Ejudge через REST API `/ej/api/v1/master/list-runs-json`.

## Требования

* Go 1.23+
* Токен с правами доступа к master API (`Authorization` заголовок).
* Базовый URL вашего сервера Ejudge (например, `https://example.com`).

## Запуск

```bash
# пример: прочитать идентификаторы контестов из файла и применить фильтр
EJUDGE_BASE_URL=https://example.com \
EJUDGE_TOKEN="Bearer <ваш_токен>" \
    go run ./main.go \
    --contest-file contests.txt \
    --filter "user_login like 'ivan%'" \
    --page-size 500
```

Флаги:

* `--base-url` — базовый URL Ejudge (можно через переменную `EJUDGE_BASE_URL`).
* `--token` — токен авторизации (можно через переменную `EJUDGE_TOKEN`).
* `--contests` — список ID контестов через запятую.
* `--contest-file` — путь к файлу с ID контестов (по одному в строке, пустые строки и строки c `#` игнорируются).
* `--contest-dir` — путь к каталогу, в котором папки названы числовыми ID контестов (например, `/home/judges`).
* `--filter` — `filter_expr`, который передаётся в Ejudge для выборки посылок.
* `--page-size` — размер страницы при пагинации (по умолчанию 200).
* `--field-mask` — необязательная маска полей для `list-runs-json`.

Выводится таблица по каждому контесту: `run_id`, пользователь, задача, статус и баллы. Строки отсортированы по времени отправки (самые новые сверху). Например:

```
Contest 42 — Algorithms 101 (runs: 3)
run_id    user   problem   status   score
123       ivan   A         OK       100
122       maria  B         WA       0
121       guest  A         CE       0
```
