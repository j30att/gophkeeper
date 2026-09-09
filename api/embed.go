// Package api содержит встроенную OpenAPI-спецификацию сервера.
package api

import _ "embed"

// OpenAPISpec содержит OpenAPI-спецификацию из api/openapi.yml.
//
//go:embed openapi.yml
var OpenAPISpec []byte
