// Package handlers содержит generated HTTP-handlers infra API.
package handlers

import (
	"bytes"
	"context"
	"strings"

	"github.com/igor/gophkeeper/api"
	"github.com/igor/gophkeeper/pkg/api/generated/infra"
)

const swaggerUIHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GophKeeper API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: "/openapi.yml",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
      });
    };
  </script>
</body>
</html>`

// Handler реализует generated infra.StrictServerInterface.
type Handler struct{}

// New создает Handler.
func New() *Handler {
	return &Handler{}
}

// Get возвращает health check.
func (h *Handler) Get(_ context.Context, _ infra.GetRequestObject) (infra.GetResponseObject, error) {
	return infra.Get200JSONResponse{Status: "ok"}, nil
}

// GetDocs возвращает Swagger UI.
func (h *Handler) GetDocs(_ context.Context, _ infra.GetDocsRequestObject) (infra.GetDocsResponseObject, error) {
	return infra.GetDocs200TexthtmlResponse{
		Body:          strings.NewReader(swaggerUIHTML),
		ContentLength: int64(len(swaggerUIHTML)),
	}, nil
}

// GetOpenapiYml возвращает OpenAPI-спецификацию.
func (h *Handler) GetOpenapiYml(
	_ context.Context,
	_ infra.GetOpenapiYmlRequestObject,
) (infra.GetOpenapiYmlResponseObject, error) {
	return infra.GetOpenapiYml200ApplicationyamlResponse{
		Body:          bytes.NewReader(api.OpenAPISpec),
		ContentLength: int64(len(api.OpenAPISpec)),
	}, nil
}
