package handlers

import (
	"context"
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
