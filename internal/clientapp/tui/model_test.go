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

func key(message string) tea.KeyMsg {
	switch message {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(message)}
	}
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

	t.Run("Должен игнорировать ошибку загрузки session", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})

		updated, cmd := model.Update(sessionLoadedMsg{err: session.ErrNotFound})
		next := updated.(Model)

		assert.Equal(t, screenAuth, next.screen)
		assert.Nil(t, cmd)
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

	t.Run("Должен показать ошибку sync", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})

		updated, cmd := model.Update(syncDoneMsg{err: errors.New("sync failed")})
		next := updated.(Model)

		assert.Nil(t, cmd)
		assert.Contains(t, next.View(), "sync failed")
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

	t.Run("Должен обработать createDoneMsg", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.token = "jwt"
		model.screen = screenCreate

		updated, cmd := model.Update(createDoneMsg{})
		next := updated.(Model)

		assert.Equal(t, screenList, next.screen)
		assert.Equal(t, "Секрет создан", next.status)
		assert.NotNil(t, cmd)
	})

	t.Run("Должен показать ошибку создания", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})

		updated, cmd := model.Update(createDoneMsg{err: errors.New("create failed")})
		next := updated.(Model)

		assert.Nil(t, cmd)
		assert.Contains(t, next.View(), "create failed")
	})

	t.Run("Должен обработать deleteDoneMsg", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.token = "jwt"

		updated, cmd := model.Update(deleteDoneMsg{})
		next := updated.(Model)

		assert.Equal(t, screenList, next.screen)
		assert.Equal(t, "Секрет удален", next.status)
		assert.NotNil(t, cmd)
	})

	t.Run("Должен показать ошибку удаления", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})

		updated, cmd := model.Update(deleteDoneMsg{err: errors.New("delete failed")})
		next := updated.(Model)

		assert.Nil(t, cmd)
		assert.Contains(t, next.View(), "delete failed")
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

		updated, _ := model.Update(key("enter"))
		next := updated.(Model)

		assert.Equal(t, screenView, next.screen)
		assert.Contains(t, next.View(), "payload")
	})

	t.Run("Должен обработать глобальные клавиши возврата", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenView

		updated, cmd := model.Update(key("q"))
		next := updated.(Model)

		assert.Nil(t, cmd)
		assert.Equal(t, screenList, next.screen)

		updated, cmd = next.Update(key("esc"))
		next = updated.(Model)

		assert.Nil(t, cmd)
		assert.Equal(t, screenList, next.screen)
	})

	t.Run("Должен завершить приложение на q из списка", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenList

		_, cmd := model.Update(key("q"))

		assert.NotNil(t, cmd)
	})

	t.Run("Должен переключать auth focus по tab и shift tab", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})

		updated, _ := model.Update(key("tab"))
		next := updated.(Model)

		assert.Equal(t, authFieldPassword, next.authFocus)

		updated, _ = next.Update(key("shift+tab"))
		next = updated.(Model)

		assert.Equal(t, authFieldLogin, next.authFocus)
	})

	t.Run("Должен переключать режим регистрации в auth", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.authFocus = authFieldMode

		updated, _ := model.Update(key("space"))
		next := updated.(Model)

		assert.True(t, next.register)
		assert.Contains(t, next.View(), "register")
	})

	t.Run("Должен вернуть auth command по enter", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.authInputs[authFieldLogin].SetValue("igor")
		model.authInputs[authFieldPassword].SetValue("password-1")

		_, cmd := model.Update(key("enter"))

		assert.NotNil(t, cmd)
	})

	t.Run("Должен открыть форму создания из списка", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenList

		updated, cmd := model.Update(key("a"))
		next := updated.(Model)

		assert.Nil(t, cmd)
		assert.Equal(t, screenCreate, next.screen)
		assert.Contains(t, next.View(), "Создать structured secret")
	})

	t.Run("Должен вернуть sync command из списка", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenList
		now := time.Now().UTC()
		model.lastSyncAt = &now

		_, cmd := model.Update(key("r"))

		assert.NotNil(t, cmd)
	})

	t.Run("Должен ничего не делать при delete без выбранного secret", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenList

		updated, cmd := model.Update(key("d"))
		next := updated.(Model)

		assert.Nil(t, cmd)
		assert.Equal(t, screenList, next.screen)
	})

	t.Run("Должен вернуть delete command для выбранного secret", func(t *testing.T) {
		secretID := uuid.New()
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenList
		model.applySync(api.SyncResponse{Secrets: []api.Secret{{ID: secretID, Name: "github"}}})

		_, cmd := model.Update(key("d"))

		assert.NotNil(t, cmd)
	})

	t.Run("Должен переключать create focus", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenCreate

		updated, _ := model.Update(key("tab"))
		next := updated.(Model)

		assert.Equal(t, createFieldName, next.createFocus)

		updated, _ = next.Update(key("shift+tab"))
		next = updated.(Model)

		assert.Equal(t, createFieldType, next.createFocus)
	})

	t.Run("Должен вернуть create command по enter", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenCreate
		model.createInputs[createFieldType].SetValue("credentials")
		model.createInputs[createFieldName].SetValue("github")

		_, cmd := model.Update(key("enter"))

		assert.NotNil(t, cmd)
	})

	t.Run("Должен показать create validation error по enter", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenCreate
		model.createInputs[createFieldType].SetValue("binary")

		updated, cmd := model.Update(key("enter"))
		next := updated.(Model)

		assert.Nil(t, cmd)
		assert.Contains(t, next.View(), "type должен быть credentials или card")
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

	t.Run("Должен вернуть пустую map при невалидном JSON value", func(t *testing.T) {
		assert.Equal(t, "{}", formatMap(map[string]interface{}{"bad": func() {}}))
	})

	t.Run("Должен отрисовать пустой selected secret", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenView

		assert.Contains(t, model.View(), "Секрет не выбран")
	})

	t.Run("Должен отрисовать blob metadata", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenView
		model.selected = &api.Secret{
			Name: "file",
			Type: api.SecretTypeBinary,
			Blob: &api.Blob{OriginalName: "doc.pdf", Size: 128},
		}

		assert.Contains(t, model.View(), "doc.pdf, 128 bytes")
	})

	t.Run("Должен отрисовать list screen", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.screen = screenList
		model.secretsList.SetSize(80, 20)

		view := model.View()

		assert.Contains(t, view, "GophKeeper secrets")
		assert.Contains(t, view, "enter - открыть")
	})

	t.Run("Должен вернуть описание и filter value для list item", func(t *testing.T) {
		now := time.Date(2026, 9, 9, 1, 2, 3, 0, time.UTC)
		item := secretListItem{secret: api.Secret{
			Name:      "github",
			Type:      api.SecretTypeCredentials,
			Version:   3,
			UpdatedAt: now,
		}}

		assert.Equal(t, "github", item.FilterValue())
		assert.Contains(t, item.Description(), "credentials")
		assert.Contains(t, item.Description(), "v3")
	})

	t.Run("Должен сбросить create form", func(t *testing.T) {
		model := New(&apiClientMock{}, &sessionStoreMock{})
		model.createInputs[createFieldName].SetValue("github")

		model.resetCreate()

		assert.Equal(t, "", model.createInputs[createFieldName].Value())
		assert.True(t, strings.Contains(model.View(), "GophKeeper"))
	})
}
