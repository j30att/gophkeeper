package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/clientapp/api"
	"github.com/igor/gophkeeper/internal/clientapp/session"
)

type apiClientMock struct {
	token       string
	syncOutput  api.SyncResponse
	created     api.Secret
	err         error
	registered  bool
	login       string
	password    string
	deletedID   uuid.UUID
	secretInput api.SecretInput
}

func (m *apiClientMock) Register(_ context.Context, login string, password string) (string, error) {
	m.registered = true
	m.login = login
	m.password = password
	return uuid.NewString(), m.err
}

func (m *apiClientMock) Login(_ context.Context, login string, password string) (string, error) {
	m.login = login
	m.password = password
	return m.token, m.err
}

func (m *apiClientMock) Sync(_ context.Context, _ string, _ *time.Time) (api.SyncResponse, error) {
	return m.syncOutput, m.err
}

func (m *apiClientMock) CreateSecret(_ context.Context, _ string, input api.SecretInput) (api.Secret, error) {
	m.secretInput = input
	return m.created, m.err
}

func (m *apiClientMock) DeleteSecret(_ context.Context, _ string, id uuid.UUID) error {
	m.deletedID = id
	return m.err
}

type sessionStoreMock struct {
	session session.Session
	err     error
	saved   session.Session
}

func (m *sessionStoreMock) Save(session session.Session) error {
	m.saved = session
	return m.err
}

func (m *sessionStoreMock) Load() (session.Session, error) {
	return m.session, m.err
}

func TestModel(t *testing.T) {
	t.Run("Должен показать auth screen", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{err: session.ErrNotFound})

		view := model.View()

		assert.Contains(t, view, "GophKeeper")
		assert.Contains(t, view, "login")
	})

	t.Run("Должен загрузить сохраненную session и выполнить sync", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{session: session.Session{Token: "jwt"}})

		message := model.Init()()
		updated, cmd := model.Update(message)
		next := updated.(Model)

		assert.Equal(t, "jwt", next.token)
		assert.Equal(t, screenList, next.screen)
		assert.NotNil(t, cmd)
	})

	t.Run("Должен обработать успешный auth", func(t *testing.T) {
		store := &sessionStoreMock{}
		model := New(&apiClientMock{}, store)

		updated, cmd := model.Update(authDoneMsg{token: "jwt"})
		next := updated.(Model)

		assert.Equal(t, "jwt", next.token)
		assert.Equal(t, screenList, next.screen)
		assert.NotNil(t, cmd)
	})

	t.Run("Должен показать auth ошибку", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})

		updated, _ := model.Update(authDoneMsg{err: errors.New("bad auth")})
		next := updated.(Model)

		assert.Contains(t, next.View(), "bad auth")
	})

	t.Run("Должен применить sync", func(t *testing.T) {
		secretID := uuid.New()
		deletedID := uuid.New()
		now := time.Now().UTC()
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenList
		model.secrets = []api.Secret{{ID: deletedID, Name: "old"}}

		updated, _ := model.Update(
			syncDoneMsg{
				response: api.SyncResponse{
					ServerTime: now,
					Secrets: []api.Secret{
						{ID: secretID, Name: "github", Type: api.SecretTypeCredentials, Version: 1, UpdatedAt: now},
					},
					Deleted: []api.DeletedSecret{{ID: deletedID, Version: 2, DeletedAt: now}},
				},
			},
		)
		next := updated.(Model)

		require.Len(t, next.secrets, 1)
		assert.Equal(t, secretID, next.secrets[0].ID)
		selected, ok := next.selectedListItem()
		require.True(t, ok)
		assert.Equal(t, "github", selected.Title())
	})

	t.Run("Должен открыть выбранный secret", func(t *testing.T) {
		secret := api.Secret{
			ID:       uuid.New(),
			Name:     "github",
			Type:     api.SecretTypeCredentials,
			Metadata: map[string]interface{}{"site": "github"},
			Payload:  map[string]interface{}{"login": "igor"},
			Version:  1,
		}
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenList
		model.applySync(api.SyncResponse{Secrets: []api.Secret{secret}})

		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		next := updated.(Model)

		assert.Equal(t, screenView, next.screen)
		assert.Contains(t, next.View(), "payload")
	})

	t.Run("Должен собрать credentials input", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.createInputs[createFieldType].SetValue("credentials")
		model.createInputs[createFieldName].SetValue("github")
		model.createInputs[createFieldMetadata].SetValue(`{"site":"github"}`)
		model.createInputs[createFieldFirst].SetValue("igor")
		model.createInputs[createFieldSecond].SetValue("secret")

		input, err := model.createSecretInput()

		require.NoError(t, err)
		assert.Equal(t, api.SecretTypeCredentials, input.Type)
		assert.Equal(t, "github", input.Name)
		assert.Equal(t, "igor", input.Payload["login"])
	})

	t.Run("Должен собрать card input", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.createInputs[createFieldType].SetValue("card")
		model.createInputs[createFieldName].SetValue("main")
		model.createInputs[createFieldFirst].SetValue("4111")
		model.createInputs[createFieldSecond].SetValue("IGOR")
		model.createInputs[createFieldThird].SetValue("12/30")
		model.createInputs[createFieldFourth].SetValue("123")

		input, err := model.createSecretInput()

		require.NoError(t, err)
		assert.Equal(t, api.SecretTypeCard, input.Type)
		assert.Equal(t, "12/30", input.Payload["expires_at"])
	})

	t.Run("Должен вернуть ошибку при неверном create type", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.createInputs[createFieldType].SetValue("binary")

		_, err := model.createSecretInput()

		require.Error(t, err)
	})

	t.Run("Должен вернуть ошибку при неверном metadata JSON", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.createInputs[createFieldType].SetValue("credentials")
		model.createInputs[createFieldMetadata].SetValue("{")

		_, err := model.createSecretInput()

		require.Error(t, err)
	})

	t.Run("Должен выполнить auth command", func(t *testing.T) {
		client := &apiClientMock{token: "jwt"}
		message := authCmd(client, "igor", "password-1", true)()

		done := message.(authDoneMsg)
		require.NoError(t, done.err)
		assert.True(t, client.registered)
		assert.Equal(t, "jwt", done.token)
	})

	t.Run("Должен выполнить sync command", func(t *testing.T) {
		now := time.Now().UTC()
		client := &apiClientMock{syncOutput: api.SyncResponse{ServerTime: now}}

		message := syncCmd(client, "jwt", nil)()

		done := message.(syncDoneMsg)
		require.NoError(t, done.err)
		assert.Equal(t, now, done.response.ServerTime)
	})

	t.Run("Должен выполнить create command", func(t *testing.T) {
		client := &apiClientMock{}

		message := createCmd(client, "jwt", api.SecretInput{Name: "github"})()

		done := message.(createDoneMsg)
		require.NoError(t, done.err)
		assert.Equal(t, "github", client.secretInput.Name)
	})

	t.Run("Должен выполнить delete command", func(t *testing.T) {
		secretID := uuid.New()
		client := &apiClientMock{}

		message := deleteCmd(client, "jwt", secretID)()

		done := message.(deleteDoneMsg)
		require.NoError(t, done.err)
		assert.Equal(t, secretID, client.deletedID)
	})

	t.Run("Должен форматировать пустую map", func(t *testing.T) {
		assert.Equal(t, "{}", formatMap(nil))
	})

	t.Run("Должен сбросить create form", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.createInputs[createFieldName].SetValue("github")

		model.resetCreate()

		assert.Equal(t, "", model.createInputs[createFieldName].Value())
		assert.True(t, strings.Contains(model.View(), "GophKeeper"))
	})
}
