package usecases

import "errors"

// ErrEmptyDependency означает, что обязательная зависимость конструктора не передана.
var ErrEmptyDependency = errors.New("empty dependency")

// ErrInvalidCredentials означает, что login или password некорректны.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrUserAlreadyExists означает, что пользователь с таким login уже существует.
var ErrUserAlreadyExists = errors.New("user already exists")

// ErrUserNotFound означает, что repository не нашел пользователя.
var ErrUserNotFound = errors.New("user not found")
