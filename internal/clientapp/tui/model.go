// Package tui содержит терминальный интерфейс клиента GophKeeper.
package tui

import (
	"context"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/igor/gophkeeper/internal/clientapp/api"
	"github.com/igor/gophkeeper/internal/clientapp/cache"
	"github.com/igor/gophkeeper/internal/clientapp/session"
)

type screen int

const (
	screenAuth screen = iota
	screenList
	screenView
	screenCreate
	screenUpdate
	screenCreateBlob
	screenDownloadBlob
	screenUpdateBlob
)

const (
	authFieldLogin = iota
	authFieldPassword
	authFieldMode
)

const (
	createFieldType = iota
	createFieldName
	createFieldMetadata
	createFieldFirst
	createFieldSecond
	createFieldThird
	createFieldFourth
)

const (
	blobFieldType = iota
	blobFieldName
	blobFieldMetadata
	blobFieldPath
	blobFieldContentType
)

const (
	downloadFieldPath = iota
)

const (
	updateBlobFieldPath = iota
	updateBlobFieldContentType
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// APIClient описывает методы HTTP-клиента, которые нужны TUI.
type APIClient interface {
	Register(ctx context.Context, login string, password string) (string, error)
	Login(ctx context.Context, login string, password string) (string, error)
	Sync(ctx context.Context, token string, since *time.Time) (api.SyncResponse, error)
	CreateSecret(ctx context.Context, token string, input api.SecretInput) (api.Secret, error)
	UpdateSecret(ctx context.Context, token string, id uuid.UUID, input api.SecretInput) (api.Secret, error)
	CreateBlobSecret(ctx context.Context, token string, input api.BlobSecretInput) (api.Secret, error)
	DownloadBlobContent(ctx context.Context, token string, id uuid.UUID, writer io.Writer) error
	UpdateBlobContent(ctx context.Context, token string, id uuid.UUID, input api.BlobContentInput) (api.Secret, error)
	DeleteSecret(ctx context.Context, token string, id uuid.UUID) error
}

// SessionStore описывает методы локального хранения сессии.
type SessionStore interface {
	Save(session.Session) error
	Load() (session.Session, error)
	Delete() error
}

// CacheStore описывает методы локального хранения cache секретов.
type CacheStore interface {
	Save(cache.Cache) error
	Load() (cache.Cache, error)
	Delete() error
}

// Model описывает состояние TUI-приложения.
type Model struct {
	apiClient APIClient
	store     SessionStore
	cache     CacheStore

	screen screen
	err    string
	status string

	token      string
	lastSyncAt *time.Time
	secrets    []api.Secret
	selected   *api.Secret

	authInputs []textinput.Model
	authFocus  int
	register   bool

	secretsList list.Model

	createInputs []textinput.Model
	createFocus  int
	createType   api.SecretType

	updateInputs []textinput.Model
	updateFocus  int

	blobInputs []textinput.Model
	blobFocus  int

	downloadInputs []textinput.Model
	downloadFocus  int

	updateBlobInputs []textinput.Model
	updateBlobFocus  int
}

// New создает Model.
func New(apiClient APIClient, store SessionStore, cacheStore ...CacheStore) Model {
	authInputs := []textinput.Model{
		newInput("login"),
		newPasswordInput("password"),
		newInput("mode"),
	}
	authInputs[0].Focus()

	createInputs := []textinput.Model{
		newInput("type"),
		newInput("name"),
		newInput(`metadata JSON, например {"site":"github"}`),
		newInput("login / card number"),
		newPasswordInput("password / holder"),
		newInput("expires_at"),
		newPasswordInput("cvv"),
	}
	createInputs[0].SetValue(string(api.SecretTypeCredentials))
	createInputs[0].Focus()

	updateInputs := []textinput.Model{
		newInput("type"),
		newInput("name"),
		newInput(`metadata JSON, например {"site":"github"}`),
		newInput("login / card number"),
		newPasswordInput("password / holder"),
		newInput("expires_at"),
		newPasswordInput("cvv"),
	}
	updateInputs[0].SetValue(string(api.SecretTypeCredentials))
	updateInputs[0].Focus()

	blobInputs := []textinput.Model{
		newInput("type: text или binary"),
		newInput("name"),
		newInput(`metadata JSON, например {"kind":"document"}`),
		newInput("file path"),
		newInput("content-type, можно пусто"),
	}
	blobInputs[0].SetValue(string(api.SecretTypeBinary))
	blobInputs[0].Focus()

	downloadInputs := []textinput.Model{
		newInput("save path, можно пусто"),
	}
	downloadInputs[0].Focus()

	updateBlobInputs := []textinput.Model{
		newInput("new file path"),
		newInput("content-type, можно пусто"),
	}
	updateBlobInputs[0].Focus()

	delegate := list.NewDefaultDelegate()
	secretsList := list.New(nil, delegate, 0, 0)
	secretsList.Title = "Secrets"
	secretsList.SetShowStatusBar(false)
	secretsList.SetFilteringEnabled(false)

	var localCache CacheStore
	if len(cacheStore) > 0 {
		localCache = cacheStore[0]
	}

	return Model{
		apiClient:        apiClient,
		store:            store,
		cache:            localCache,
		screen:           screenAuth,
		authInputs:       authInputs,
		createInputs:     createInputs,
		createType:       api.SecretTypeCredentials,
		updateInputs:     updateInputs,
		blobInputs:       blobInputs,
		downloadInputs:   downloadInputs,
		updateBlobInputs: updateBlobInputs,
		secretsList:      secretsList,
	}
}

// Init запускает начальную загрузку сессии.
func (m Model) Init() tea.Cmd {
	return loadSessionCmd(m.store)
}
