# GophKeeper

GophKeeper - учебный клиент-серверный менеджер приватных данных.

Сейчас реализуется серверная часть: регистрация, логин, хранение structured-секретов, blob-секретов, Swagger/OpenAPI и Postgres storage.
Синхронизация и клиентская TUI-часть отложены до стабилизации backend API.

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
