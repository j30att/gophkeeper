package tui

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/igor/gophkeeper/internal/clientapp/cache"
)

// Update обрабатывает TUI-события.
func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.secretsList.SetSize(msg.Width, msg.Height-4)
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	case sessionLoadedMsg:
		if msg.err == nil {
			m.token = msg.token
			m.screen = screenList
			return m, loadCacheCmd(m.cache)
		}
		return m, nil
	case cacheLoadedMsg:
		if msg.err != nil && !errors.Is(msg.err, cache.ErrNotFound) {
			m.err = msg.err.Error()
		}
		if msg.err == nil {
			m.secrets = msg.cache.Secrets
			m.lastSyncAt = msg.cache.LastSyncAt
			refreshSecretsList(&m)
			if len(m.secrets) > 0 {
				m.status = "Загружен локальный cache"
			}
		}
		return m, syncCmd(m.apiClient, m.token, m.lastSyncAt)
	case cacheSavedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		}
		return m, nil
	case authDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = "Вход выполнен"
		m.token = msg.token
		m.screen = screenList
		return m, saveSessionCmd(m.store, msg.token, syncCmd(m.apiClient, msg.token, nil))
	case syncDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = "Список обновлен"
		m.lastSyncAt = &msg.response.ServerTime
		applySync(&m, msg.response)
		return m, saveCacheCmd(m.cache, cache.Cache{LastSyncAt: m.lastSyncAt, Secrets: m.secrets})
	case createDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = "Секрет создан"
		m.screen = screenList
		return m, syncCmd(m.apiClient, m.token, nil)
	case updateDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = "Секрет обновлен"
		m.screen = screenList
		return m, syncCmd(m.apiClient, m.token, nil)
	case createBlobDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = "Blob-секрет создан"
		m.screen = screenList
		return m, syncCmd(m.apiClient, m.token, nil)
	case downloadBlobDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = "Blob сохранен: " + msg.path
		m.screen = screenView
		return m, nil
	case updateBlobDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = "Blob-содержимое заменено"
		m.screen = screenList
		return m, syncCmd(m.apiClient, m.token, nil)
	case logoutDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = "Выход выполнен"
		m.token = ""
		m.lastSyncAt = nil
		m.secrets = nil
		m.selected = nil
		refreshSecretsList(&m)
		m.screen = screenAuth
		return m, nil
	case deleteDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = "Секрет удален"
		m.screen = screenList
		return m, syncCmd(m.apiClient, m.token, nil)
	}
	return m.updateCurrent(message)
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.screen == screenAuth || m.screen == screenList {
			return m, tea.Quit
		}
		m.screen = screenList
		return m, nil
	case "esc":
		m.screen = screenList
		return m, nil
	}
	return m.updateCurrent(msg)
}

func (m Model) updateCurrent(message tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenAuth:
		return m.updateAuth(message)
	case screenList:
		return m.updateList(message)
	case screenView:
		return m.updateView(message)
	case screenCreate:
		return m.updateCreate(message)
	case screenUpdate:
		return m.updateUpdate(message)
	case screenCreateBlob:
		return m.updateCreateBlob(message)
	case screenDownloadBlob:
		return m.updateDownloadBlob(message)
	case screenUpdateBlob:
		return m.updateUpdateBlob(message)
	}
	return m, nil
}

func (m Model) updateAuth(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "tab", "shift+tab":
			m.authInputs[m.authFocus].Blur()
			if msg.String() == "shift+tab" {
				m.authFocus--
				if m.authFocus < 0 {
					m.authFocus = authFieldMode
				}
			} else {
				m.authFocus++
				if m.authFocus > authFieldMode {
					m.authFocus = 0
				}
			}
			m.authInputs[m.authFocus].Focus()
			return m, nil
		case " ":
			if m.authFocus == authFieldMode {
				m.register = !m.register
				return m, nil
			}
		case "enter":
			return m, authCmd(m.apiClient, m.authInputs[0].Value(), m.authInputs[1].Value(), m.register)
		}
	}
	var cmd tea.Cmd
	m.authInputs[m.authFocus], cmd = m.authInputs[m.authFocus].Update(message)
	return m, cmd
}

func (m Model) updateList(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "r":
			return m, syncCmd(m.apiClient, m.token, m.lastSyncAt)
		case "a":
			resetCreate(&m)
			m.screen = screenCreate
			return m, nil
		case "b":
			resetCreateBlob(&m)
			m.screen = screenCreateBlob
			return m, nil
		case "d":
			selected, ok := m.selectedListItem()
			if !ok {
				return m, nil
			}
			return m, deleteCmd(m.apiClient, m.token, selected.secret.ID)
		case "l":
			return m, logoutCmd(m.store, m.cache)
		case "enter":
			selected, ok := m.selectedListItem()
			if !ok {
				return m, nil
			}
			secret := selected.secret
			m.selected = &secret
			m.screen = screenView
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.secretsList, cmd = m.secretsList.Update(message)
	return m, cmd
}

func (m Model) updateView(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "e":
			if m.selected == nil || m.selected.Payload == nil {
				m.err = "выбранный секрет не является structured"
				return m, nil
			}
			resetUpdate(&m)
			m.screen = screenUpdate
			return m, nil
		case "s":
			if m.selected == nil || m.selected.Blob == nil {
				m.err = "выбранный секрет не является blob"
				return m, nil
			}
			resetDownloadBlob(&m)
			m.screen = screenDownloadBlob
			return m, nil
		case "p":
			if m.selected == nil || m.selected.Blob == nil {
				m.err = "выбранный секрет не является blob"
				return m, nil
			}
			resetUpdateBlob(&m)
			m.screen = screenUpdateBlob
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateCreate(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "tab", "shift+tab":
			m.createInputs[m.createFocus].Blur()
			if msg.String() == "shift+tab" {
				m.createFocus--
				if m.createFocus < 0 {
					m.createFocus = len(m.createInputs) - 1
				}
			} else {
				m.createFocus++
				if m.createFocus >= len(m.createInputs) {
					m.createFocus = 0
				}
			}
			m.createInputs[m.createFocus].Focus()
			return m, nil
		case "enter":
			input, err := m.createSecretInput()
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			return m, createCmd(m.apiClient, m.token, input)
		}
	}
	var cmd tea.Cmd
	m.createInputs[m.createFocus], cmd = m.createInputs[m.createFocus].Update(message)
	return m, cmd
}

func (m Model) updateUpdate(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "tab", "shift+tab":
			m.updateInputs[m.updateFocus].Blur()
			if msg.String() == "shift+tab" {
				m.updateFocus--
				if m.updateFocus < 0 {
					m.updateFocus = len(m.updateInputs) - 1
				}
			} else {
				m.updateFocus++
				if m.updateFocus >= len(m.updateInputs) {
					m.updateFocus = 0
				}
			}
			m.updateInputs[m.updateFocus].Focus()
			return m, nil
		case "enter":
			if m.selected == nil || m.selected.Payload == nil {
				m.err = "выбранный секрет не является structured"
				return m, nil
			}
			input, err := m.updateSecretInput()
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			return m, updateCmd(m.apiClient, m.token, m.selected.ID, input)
		}
	}
	var cmd tea.Cmd
	m.updateInputs[m.updateFocus], cmd = m.updateInputs[m.updateFocus].Update(message)
	return m, cmd
}

func (m Model) updateCreateBlob(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "tab", "shift+tab":
			m.blobInputs[m.blobFocus].Blur()
			if msg.String() == "shift+tab" {
				m.blobFocus--
				if m.blobFocus < 0 {
					m.blobFocus = len(m.blobInputs) - 1
				}
			} else {
				m.blobFocus++
				if m.blobFocus >= len(m.blobInputs) {
					m.blobFocus = 0
				}
			}
			m.blobInputs[m.blobFocus].Focus()
			return m, nil
		case "enter":
			input, close, err := m.blobSecretInput()
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			return m, createBlobCmd(m.apiClient, m.token, input, close)
		}
	}
	var cmd tea.Cmd
	m.blobInputs[m.blobFocus], cmd = m.blobInputs[m.blobFocus].Update(message)
	return m, cmd
}

func (m Model) updateDownloadBlob(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok && msg.String() == "enter" {
		if m.selected == nil || m.selected.Blob == nil {
			m.err = "выбранный секрет не является blob"
			return m, nil
		}
		path, file, err := m.downloadBlobOutput()
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		return m, downloadBlobCmd(m.apiClient, m.token, m.selected.ID, path, file.Close, file)
	}
	var cmd tea.Cmd
	m.downloadInputs[m.downloadFocus], cmd = m.downloadInputs[m.downloadFocus].Update(message)
	return m, cmd
}

func (m Model) updateUpdateBlob(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "tab", "shift+tab":
			m.updateBlobInputs[m.updateBlobFocus].Blur()
			if msg.String() == "shift+tab" {
				m.updateBlobFocus--
				if m.updateBlobFocus < 0 {
					m.updateBlobFocus = len(m.updateBlobInputs) - 1
				}
			} else {
				m.updateBlobFocus++
				if m.updateBlobFocus >= len(m.updateBlobInputs) {
					m.updateBlobFocus = 0
				}
			}
			m.updateBlobInputs[m.updateBlobFocus].Focus()
			return m, nil
		case "enter":
			if m.selected == nil || m.selected.Blob == nil {
				m.err = "выбранный секрет не является blob"
				return m, nil
			}
			input, close, err := m.updateBlobInput()
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			input.ExpectedVersion = m.selected.Version
			return m, updateBlobCmd(m.apiClient, m.token, m.selected.ID, input, close)
		}
	}
	var cmd tea.Cmd
	m.updateBlobInputs[m.updateBlobFocus], cmd = m.updateBlobInputs[m.updateBlobFocus].Update(message)
	return m, cmd
}
