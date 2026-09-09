# GophKeeper

GophKeeper - учебный клиент-серверный менеджер приватных данных.

В репозитории есть сервер и CLI-клиент с TUI. Сервер отвечает за регистрацию, логин, хранение structured-секретов, blob-секретов, Swagger/OpenAPI, Postgres storage и синхронизацию. Клиент умеет логиниться, регистрироваться, синхронизировать список секретов, кешировать локальный snapshot, просматривать structured/blob-секреты, создавать credentials/card/blob, скачивать и заменять blob-содержимое, редактировать и удалять секреты.

## Требования

- Go 1.25+
- Docker и Docker Compose
- `oapi-codegen`, если нужно регенерировать API-код

## Быстрый запуск

Поднять Postgres:

```bash
make compose-up
```

Применить миграции:

```bash
make migrate-up
```

Запустить сервер:

```bash
make run-server-dev
```

По умолчанию сервер слушает `127.0.0.1:8080`.

Swagger UI:

```text
http://127.0.0.1:8080/docs
```

OpenAPI spec:

```text
http://127.0.0.1:8080/openapi.yml
```

## Проверки

Сгенерировать API-код:

```bash
make generate-api
```

Запустить тесты с race и coverage:

```bash
make test
```

Собрать сервер:

```bash
make build-server
```

Собрать клиент:

```bash
make build-client
```

Версия клиента и дата сборки:

```bash
./bin/gophkeeper-client version
```

Запустить TUI-клиент:

```bash
./bin/gophkeeper-client --server http://127.0.0.1:8080
```

Для dev-запуска клиента без сборки:

```bash
make run-client-dev
```

Основные клавиши TUI:

- `space` на экране входа переключает login/register;
- `tab` переключает поля ввода;
- `enter` выполняет действие или открывает выбранный секрет;
- `a` открывает форму создания structured-секрета;
- `b` открывает форму создания blob-секрета из файла;
- `e` на экране structured-секрета открывает редактирование;
- `r` синхронизирует список с сервером;
- `d` удаляет выбранный секрет;
- `l` выполняет logout и удаляет локальные `session.json`/`secrets.json`;
- `s` на экране blob-секрета скачивает файл;
- `p` на экране blob-секрета заменяет файл;
- `esc` возвращает к списку;
- `q` выходит или возвращает из просмотра к списку.

Собрать cleanup-команду:

```bash
make build-cleanup
```

Удалить с диска blob-файлы, которые уже удалены логически:

```bash
make cleanup-blobs
```

## Примеры curl

Health check:

```bash
curl -i http://127.0.0.1:8080/
```

Регистрация:

```bash
curl -i -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"login":"igor","password":"password-1"}'
```

Логин:

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"login":"igor","password":"password-1"}' | jq -r '.token')
```

Создать structured-секрет:

```bash
curl -i -X POST http://127.0.0.1:8080/api/v1/secrets \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{
    "type": "credentials",
    "name": "github",
    "metadata": {"site": "github.com"},
    "payload": {"login": "igor", "password": "secret"}
  }'
```

Создать секрет банковской карты:

```bash
curl -i -X POST http://127.0.0.1:8080/api/v1/secrets \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{
    "type": "card",
    "name": "main card",
    "metadata": {"bank": "demo"},
    "payload": {
      "number": "4111111111111111",
      "holder": "IGOR TEST",
      "expires_at": "12/30",
      "cvv": "123"
    }
  }'
```

Получить список секретов:

```bash
curl -s http://127.0.0.1:8080/api/v1/secrets \
  -H "Authorization: Bearer ${TOKEN}"
```

Получить изменения для синхронизации:

```bash
curl -s 'http://127.0.0.1:8080/api/v1/sync?since=2026-09-08T00:00:00Z' \
  -H "Authorization: Bearer ${TOKEN}"
```

Если `since` не передавать, сервер вернет полный snapshot активных секретов:

```bash
curl -s http://127.0.0.1:8080/api/v1/sync \
  -H "Authorization: Bearer ${TOKEN}"
```

Обновить structured-секрет:

```bash
curl -i -X PUT http://127.0.0.1:8080/api/v1/secrets/<secret_id> \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{
    "type": "credentials",
    "name": "github",
    "expected_version": 1,
    "metadata": {"site": "github.com"},
    "payload": {"login": "igor", "password": "new-secret"}
  }'
```

Создать blob-секрет:

```bash
curl -i -X POST http://127.0.0.1:8080/api/v1/secrets/blob \
  -H "Authorization: Bearer ${TOKEN}" \
  -F 'type=text' \
  -F 'name=big-note' \
  -F 'metadata={"kind":"note"}' \
  -F 'file=@./note.txt;type=text/plain'
```

Скачать содержимое blob-секрета:

```bash
curl -L http://127.0.0.1:8080/api/v1/secrets/<secret_id>/content \
  -H "Authorization: Bearer ${TOKEN}" \
  -o downloaded.bin
```

Заменить содержимое blob-секрета:

```bash
curl -i -X PUT http://127.0.0.1:8080/api/v1/secrets/<secret_id>/content \
  -H "Authorization: Bearer ${TOKEN}" \
  -F 'expected_version=1' \
  -F 'file=@./new-note.txt;type=text/plain'
```

Удалить секрет:

```bash
curl -i -X DELETE http://127.0.0.1:8080/api/v1/secrets/<secret_id> \
  -H "Authorization: Bearer ${TOKEN}"
```
