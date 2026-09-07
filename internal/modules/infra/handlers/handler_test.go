package handlers

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/pkg/api/generated/infra"
)

func TestHandlerGet(t *testing.T) {
	handler := New()

	response, err := handler.Get(context.Background(), infra.GetRequestObject{})

	require.NoError(t, err)
	okResponse, ok := response.(infra.Get200JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "ok", okResponse.Status)
}

func TestHandlerGetDocs(t *testing.T) {
	handler := New()

	response, err := handler.GetDocs(context.Background(), infra.GetDocsRequestObject{})

	require.NoError(t, err)
	okResponse, ok := response.(infra.GetDocs200TexthtmlResponse)
	require.True(t, ok)
	body, err := io.ReadAll(okResponse.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "SwaggerUIBundle")
}

func TestHandlerGetOpenapiYml(t *testing.T) {
	handler := New()

	response, err := handler.GetOpenapiYml(context.Background(), infra.GetOpenapiYmlRequestObject{})

	require.NoError(t, err)
	okResponse, ok := response.(infra.GetOpenapiYml200ApplicationyamlResponse)
	require.True(t, ok)
	body, err := io.ReadAll(okResponse.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "openapi: 3.0.3")
}
