# GophKeeper: рабочее ТЗ и план реализации

Документ фиксирует договоренности по проекту GophKeeper и будет обновляться по мере уточнений.
За основу архитектуры берем `/Users/igor/projects/x5/odp-sync`: похожий layout, OpenAPI-first подход, генерация серверного кода через `oapi-codegen`, модульная сборка зависимостей, handlers/usecases/repositories, Makefile и тестовый контур.

Рабочий `paas`-фреймворк из `odp-sync` не используем. Вместо него делаем маленький локальный аналог для bootstrap: `App`, `Builder`, `Container`, lifecycle, config, logger, HTTP server и компоненты инфраструктуры.

Предварительный Go module path: `github.com/igor/gophkeeper`.
Если после создания GitHub-репозитория путь будет другим, поменяем `go.mod` и internal imports механической правкой.

## Цель проекта

GophKeeper - клиент-серверный менеджер приватных данных.

Система должна позволять пользователю:

- зарегистрироваться;
- войти в систему;
- хранить приватные записи разных типов;
- синхронизировать записи между несколькими клиентами одного пользователя;
- получать свои записи по запросу;
- узнать версию и дату сборки CLI-клиента.

В одной репе будут две аппы:

- сервер: HTTP API, авторизация, хранение, синхронизация;
- клиент: CLI-приложение для Windows, Linux и macOS.

Начинаем с сервера. Клиент проектируем позже отдельным этапом после стабилизации API и модели синхронизации.

## Что берем из `odp-sync`

Сохраняем:

- структуру `cmd/<app>/main.go`;
- `internal/bootstrap` как место сборки приложения;
- модульный подход `internal/modules/<module>`;
- разделение на `handlers`, `usecases`, `repositories`;
- интерфейсы зависимостей рядом с потребителем;
- generated API в `pkg/api/generated`;
- `api/openapi.yml`;
- отдельные `api/configs/*.yml` для генерации по тегам;
- `api/embed.go` для отдачи OpenAPI spec;
- Swagger UI на `/docs`;
- health check на `/`;
- Makefile targets: `generate-api`, `build`, `test`, `lint`, `fmt`, `run-dev`, `mock-generate`;
- mockery/testify для unit-тестов;
- `go-chi/chi` для HTTP роутинга;
- `go-playground/validator` для validation tags из OpenAPI;
- `zerolog` для логирования;
- `pgx` для Postgres;
- `golang-migrate` или аналогичный CLI-подход для миграций;
- `oapi-codegen` strict-server для бизнес handlers.

Не берем:

- `gitlab.x5food.tech/golang/paas/v2`;
- рабочие компоненты paas logger/config/httpserver/redis/etc.;
- Greenplum и зависимость `odp-inbox-proxy`;
- рабочие CI/approvers/dashboard файлы, если они не нужны локально.

## Предлагаемый layout

```text
.
├── api
│   ├── configs
│   │   ├── auth.yml
│   │   ├── docs.yml
│   │   ├── infra.yml
│   │   └── secrets.yml
│   ├── embed.go
│   └── openapi.yml
├── cmd
│   ├── gophkeeper-client
│   │   └── main.go
│   └── gophkeeper-server
│       └── main.go
├── configs
│   ├── config.dev.yml
│   └── config.prod.yml
├── internal
│   ├── app
│   │   ├── app.go
│   │   ├── builder.go
│   │   ├── component.go
│   │   └── container.go
│   ├── bootstrap
│   │   ├── bootstrap.go
│   │   ├── config.go
│   │   ├── documentation.go
│   │   ├── infrastructure.go
│   │   └── start.go
│   ├── components
│   │   ├── logger
│   │   └── postgres
│   ├── config
│   ├── crypto
│   ├── errors
│   │   └── httperror
│   ├── middleware
│   │   ├── auth.go
│   │   ├── request_id.go
│   │   └── recovery.go
│   └── modules
│       ├── auth
│       │   ├── handlers
│       │   ├── repositories/postgres
│       │   └── usecases
│       └── secrets
│           ├── handlers
│           ├── repositories/postgres
│           └── usecases
├── migrations
├── pkg
│   └── api/generated
├── Makefile
├── go.mod
└── README.md
```

`internal/app` - наш mini-paas. Он должен закрыть только нужный минимум:

- создание приложения через builder;
- регистрация компонентов;
- доступ к logger/config/postgres через container;
- старт HTTP server;
- graceful shutdown;
- подключение workers, если понадобятся позже.

## Сервер: доменная модель

### Пользователь

Минимальная модель:

- `id UUID`;
- `login`;
- `password_hash`;
- `created_at`;
- `updated_at`.

Пароль пользователя хранится только как hash.
Предлагаемый алгоритм: `argon2id` или `bcrypt`.

### Секрет

Секрет - это доменная пользовательская запись. Физическое хранение крупных данных отделяем от доменной таблицы.

Таблица `secrets`:

- `id UUID`;
- `user_id UUID`;
- `type`;
- `name`;
- `metadata`;
- `payload`;
- `blob_id`;
- `version`;
- `created_at`;
- `updated_at`;
- `deleted_at`;

Типы:

- `credentials` - логин/пароль;
- `text` - произвольный текст;
- `binary` - произвольные бинарные данные;
- `card` - банковская карта;
- позднее опционально `otp`.

Гибридное хранение:

- `credentials` и `card` - структурные секреты, храним в `secrets.payload` как JSONB;
- `text` и `binary` - blob-секреты, содержимое храним отдельно как файл;
- `secrets.metadata` - пользовательская метаинформация для всех типов;
- `secrets.blob_id` - ссылка на физический blob для `text` и `binary`.

Причина: произвольный текст может быть очень большим, а произвольные бинарные данные фактически означают любой файл. Не кладем такие payload'ы в строку `secrets`, чтобы не раздувать таблицу и не гонять крупные данные через JSON.

### Blob

Blob - это физический объект хранения для `text` и `binary`.

Таблица `blobs`:

- `id UUID`;
- `user_id UUID`;
- `original_name`;
- `storage_name`;
- `storage_path`;
- `content_type`;
- `size`;
- `checksum_sha256`;
- `encryption_nonce`;
- `created_at`;
- `deleted_at`.

При загрузке файла сервер:

- принимает содержимое как stream или multipart file;
- генерирует новое внутреннее имя файла;
- сохраняет файл в blob storage;
- считает размер и checksum;
- сохраняет метаданные физического объекта в `blobs`;
- создает или обновляет доменную запись в `secrets`, связанную через `blob_id`.

При выдаче пользователю сервер "склеивает" ответ:

- из `secrets` берет доменную часть: `id`, `type`, `name`, пользовательскую `metadata`, `version`;
- из `blobs` берет файловую мету: исходное имя, MIME type, размер, checksum;
- из blob storage читает содержимое по `storage_path`.

Для MVP blob storage - локальная файловая система через интерфейс `BlobStorage`.
Позже реализацию можно заменить на MinIO/S3 без изменения usecase-слоя.

Шифрование для учебного проекта делаем на сервере.

- сервер хранит зашифрованный payload и metadata;
- для blob-секретов шифрует файл перед записью в storage;
- используем AES-GCM;
- encryption key берем из server config/env как обычную строку;
- внутри приложения получаем AES-256 key через SHA-256 от этой строки;
- для каждой structured-записи и каждого blob-файла генерируем отдельный nonce;
- nonce храним рядом с данными: для structured payload в `secrets`, для файлов в `blobs.encryption_nonce`;
- в коде все равно закладываем интерфейс `Encryptor`, чтобы позже можно было заменить реализацию без переписывания usecase-слоя.
- в коде оставляем комментарий, что для production лучше использовать случайный 32-byte ключ в base64 из secret storage, а не человекочитаемую строку.

## Сервер: API

OpenAPI spec будет основным контрактом.

Предлагаемые endpoint'ы MVP:

```text
GET  /                         health check
GET  /docs                     Swagger UI
GET  /openapi.yml              OpenAPI spec

POST /api/v1/auth/register     регистрация
POST /api/v1/auth/login        логин, выдача токена

POST /api/v1/secrets           создать structured-запись credentials/card
POST /api/v1/secrets/blob      создать blob-запись text/binary
GET  /api/v1/secrets           список записей пользователя
GET  /api/v1/secrets/{id}      получить метаданные записи
GET  /api/v1/secrets/{id}/content
                              получить содержимое blob-записи
PUT  /api/v1/secrets/{id}      обновить structured-запись или метаданные
PUT  /api/v1/secrets/{id}/content
                              заменить содержимое blob-записи
DELETE /api/v1/secrets/{id}    soft-delete

POST /api/v1/sync              синхронизация изменений
```

Для `text` и `binary` не используем JSON payload с base64.
Загрузка идет через `multipart/form-data`, чтобы API нормально отображался в Swagger и клиент мог передавать файл stream'ом.

Поля `multipart/form-data` для blob upload:

- `type` - `text` или `binary`;
- `name` - пользовательское название секрета;
- `metadata` - JSON-строка с пользовательской метаинформацией;
- `file` - содержимое файла stream'ом.

Скачивание blob-содержимого идет через `GET /api/v1/secrets/{id}/content` stream-ответом.

Авторизация:

- `Authorization: Bearer <token>`;
- MVP: JWT access token;
- refresh token не делаем на первом этапе.

Синхронизация:

- у каждой записи есть `version` и `updated_at`;
- клиент отправляет локальные изменения и известную server revision;
- сервер возвращает актуальные изменения;
- конфликт MVP: явный `409 Conflict` при несовпадении `version`.

Рекомендуемый MVP: `version` + `409 Conflict` для точечных update, а sync пока возвращает server-side изменения с момента `since`.

## Сервер: модули

### `auth`

Use cases:

- `Register`;
- `Login`;
- `ValidateToken` или отдельный token service.

Repository:

- создание пользователя;
- поиск пользователя по login;
- поиск пользователя по id.

Handlers:

- generated strict handlers из `pkg/api/generated/auth`;
- validation request body;
- mapping domain errors в HTTP ошибки.

### `secrets`

Use cases:

- `CreateSecret`;
- `GetSecret`;
- `ListSecrets`;
- `UpdateSecret`;
- `DeleteSecret`;
- `SyncSecrets`.

Repository:

- CRUD по `user_id`;
- optimistic update по `version`;
- выборка изменений после timestamp/revision;
- soft delete.

Handlers:

- generated strict handlers из `pkg/api/generated/secrets`;
- user id берется из auth middleware context.

## Конфиг

Предлагаемый `configs/config.dev.yml`:

```yaml
server:
  host: "127.0.0.1"
  port: 8080
  shutdown_timeout: "10s"

logger:
  level: "debug"
  pretty: true

postgres:
  dsn: "postgres://gophkeeper:gophkeeper@localhost:5432/gophkeeper?sslmode=disable"
  max_conns: 10
  min_conns: 1
  query_timeout: "5s"

auth:
  jwt_secret: "dev-secret"
  access_token_ttl: "24h"

crypto:
  master_key: "dev-32-byte-key-change-before-prod"
```

Загрузка конфига:

- путь из env `CONFIG_PATH`;
- fallback на `configs/config.dev.yml`;
- env overrides можно добавить позже.

## База данных

Postgres берем как основной storage для пользователей, доменных записей и метаданных blob-объектов.
Крупное содержимое `text` и `binary` хранится в blob storage.
Локально Postgres поднимается через Docker Compose.

Предварительные миграции:

- `users`;
- `secrets`;
- `blobs`;
- индексы по `users.login`, `secrets.user_id`, `secrets.updated_at`, `blobs.user_id`;
- unique/foreign keys;
- optimistic lock через `version`.

Связи:

- `secrets.user_id -> users.id`;
- `secrets.blob_id -> blobs.id`;
- `blobs.user_id -> users.id`.

Для `secrets` нужна проверка на уровне приложения:

- `credentials/card` должны иметь `payload` и не должны иметь `blob_id`;
- `text/binary` должны иметь `blob_id` и не должны иметь большой `payload`;
- `metadata` допускается для всех типов.

Для локальной разработки нужен `docker-compose.yml` с Postgres.
Сервер должен подниматься против локальной базы без ручной настройки окружения.

Минимальный compose:

- сервис `postgres`;
- база `gophkeeper`;
- пользователь `gophkeeper`;
- пароль `gophkeeper`;
- port mapping `5432:5432`;
- volume для сохранения данных между перезапусками.

## Клиент

Пока не реализуем, но фиксируем требования:

- CLI binary для Windows/Linux/macOS;
- основной пользовательский интерфейс - TUI;
- команды `version`, `register`, `login`, `secrets list/get/create/update/delete`, `sync`;
- локальное хранение токена и, возможно, кеша секретов;
- сборочные флаги для version/build date.

Выбор для клиента:

- `bubbletea` - TUI framework;
- `bubbles` - готовые TUI-компоненты;
- `lipgloss` - стилизация TUI;
- стандартный пакет `flag` - для служебных параметров и запуска TUI.

Ожидаемые TUI-экраны:

- экран логина/регистрации;
- список секретов;
- просмотр секрета;
- форма создания/редактирования structured-секрета;
- загрузка/скачивание blob-секрета;
- экран синхронизации и ошибок конфликтов.

Команда `version` должна работать без запуска TUI.

## Тесты и качество

Требование: unit coverage не меньше 70%.
На первом проходе пишем unit-тесты на весь реализуемый код.
Интеграционные и функциональные тесты рассматриваем позже отдельным этапом.

Минимальный test strategy:

- unit-тесты handlers с mocked usecases;
- unit-тесты usecases с mocked repositories;
- unit-тесты repositories через mock интерфейсов `pgx`/storage;
- tests для auth/token/password hashing;
- tests для crypto encrypt/decrypt;
- tests для config validation;
- tests для middleware auth/request id/recovery.

Что тестируем unit-тестами обязательно:

- constructors и nil dependency validation;
- happy path каждого handler/usecase/repository метода;
- validation errors;
- domain errors и HTTP mapping;
- authorization checks;
- optimistic lock/version conflicts;
- encryption/decryption ошибок;
- blob metadata + storage interaction;
- soft delete behavior.

Не делаем на первом проходе:

- testcontainers;
- полноценные end-to-end сценарии;
- нагрузочные тесты;
- TUI snapshot/golden тесты, пока не начнем клиент.

Инструменты:

- `testing`;
- `testify`;
- `mockery`;
- `go test -race -coverprofile`;
- `golangci-lint`;
- `gofumpt`/`goimports`.

Каждый экспортированный пакет, тип, функция и переменная должны иметь Go doc comment.
Это надо учитывать сразу, иначе в конце будет дорогая правка.

## Makefile

Нужные targets:

- `make all` - generate, build, fmt, lint, test;
- `make generate-api`;
- `make build-server`;
- `make build-client`;
- `make build-all`;
- `make run-server`;
- `make run-server-dev`;
- `make test`;
- `make lint`;
- `make fmt`;
- `make deps`;
- `make migrate-up`;
- `make migrate-down`;
- `make migrate-create NAME=...`;
- `make mock-generate`;
- `make clean`;
- `make version`.

Версия клиента:

- `VERSION`;
- `BUILD_DATE`;
- ldflags: `-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)`.

## Этапы работ

### Этап 1. Скелет репозитория

- Инициализировать Go module.
- Создать layout.
- Добавить mini-paas skeleton.
- Добавить config loader.
- Добавить logger.
- Добавить HTTP server lifecycle.
- Добавить health check, `/docs`, `/openapi.yml`.
- Добавить Makefile.
- Добавить `docker-compose.yml` с Postgres.
- Добавить OpenAPI генерацию.
- Проверка: `make generate-api`, `make build-server`, `make test`.

### Этап 2. Auth module

- Описать auth endpoints в OpenAPI.
- Сгенерировать код.
- Создать migrations для `users`.
- Реализовать password hashing.
- Реализовать JWT service.
- Реализовать register/login handlers/usecases/repository.
- Добавить auth middleware.
- Покрыть unit-тестами.

### Этап 3. Secrets module

- Описать secrets endpoints в OpenAPI.
- Сгенерировать код.
- Создать migrations для `secrets`.
- Реализовать CRUD.
- Реализовать encrypt/decrypt layer.
- Реализовать optimistic lock через `version`.
- Покрыть unit-тестами.

### Этап 4. Sync MVP

- Уточнить контракт sync.
- Реализовать получение изменений с `since`.
- Реализовать отправку локальных изменений.
- Реализовать конфликтную стратегию.
- Добавить тесты на сценарии нескольких клиентов.

### Этап 5. CLI client MVP

- Выбрать CLI framework.
- Реализовать version/build date.
- Реализовать register/login.
- Реализовать CRUD команды.
- Реализовать базовый sync.
- Настроить cross-platform build.

### Этап 6. Документация и доведение

- README с запуском сервера и клиента.
- Swagger актуален.
- Комментарии на экспортированные сущности.
- Coverage >= 70%.
- Линтер проходит.

## Уточняющие вопросы перед реализацией

Открытых архитектурных вопросов для старта серверной реализации нет.

## Текущие решения по умолчанию

Если не переопределим:

- Go server на `chi`;
- Go module path предварительно `github.com/igor/gophkeeper`;
- OpenAPI + `oapi-codegen` strict-server;
- Postgres + `pgx`;
- `docker-compose.yml` с Postgres для локальной разработки;
- migrations через `golang-migrate`;
- auth через JWT;
- refresh token в MVP не делаем;
- password hash через `argon2id`;
- payload, metadata и blob-файлы шифруем AES-GCM на сервере в MVP;
- AES-256 key получаем через SHA-256 от строки из config/env;
- `text` и `binary` храним как blob-файлы с метаданными в таблице `blobs`;
- blob upload делаем через `multipart/form-data`;
- `credentials` и `card` храним как JSONB payload в `secrets`;
- sync через `version` и `409 Conflict`;
- клиент делаем как TUI на `bubbletea`;
- для запуска клиента используем стандартный пакет `flag`, без `cobra`;
- команда `version` работает без запуска TUI;
- тесты через `testing`, `testify`, `mockery`.
- на первом проходе пишем только unit-тесты, интеграционные тесты откладываем.
