package usecases

import "errors"

// ErrEmptyDependency означает, что обязательная зависимость конструктора не передана.
var ErrEmptyDependency = errors.New("empty dependency")

// ErrInvalidSecretType означает, что тип секрета не поддерживается JSON CRUD.
var ErrInvalidSecretType = errors.New("invalid secret type")

// ErrSecretNotFound означает, что secret не найден.
var ErrSecretNotFound = errors.New("secret not found")

// ErrBlobNotFound означает, что blob-файл не найден.
var ErrBlobNotFound = errors.New("blob not found")

// ErrInvalidJSON означает, что JSON-поле некорректно.
var ErrInvalidJSON = errors.New("invalid json")

// ErrEmptyContent означает, что содержимое blob-секрета не передано.
var ErrEmptyContent = errors.New("empty content")
