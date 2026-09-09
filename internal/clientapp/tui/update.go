package tui

import (
	"errors"
	"net/http"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/igor/gophkeeper/internal/clientapp/api"
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
			m.setScreen(screenList)
			return m, loadCacheCmd(m.cache)
		}
		return m, nil
	case cacheLoadedMsg:
		if msg.err != nil && !errors.Is(msg.err, cache.ErrNotFound) {
			m.setError(msg.err.Error())
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
			m.setError(msg.err.Error())
		}
		return m, nil
	case authDoneMsg:
		if msg.err != nil {
			m.setError(msg.err.Error())
			return m, nil
		}
		m.err = ""
		m.status = "Вход выполнен"
		m.token = msg.token
		m.setScreen(screenList)
		return m, saveSessionCmd(m.store, msg.token, syncCmd(m.apiClient, msg.token, nil))
	case syncDoneMsg:
		if msg.err != nil {
			m.setError(msg.err.Error())
			return m, nil
		}
		m.err = ""
		m.status = "Список обновлен"
		m.lastSyncAt = &msg.response.ServerTime
		applySync(&m, msg.response)
		return m, saveCacheCmd(m.cache, cache.Cache{LastSyncAt: m.lastSyncAt, Secrets: m.secrets})
	case createDoneMsg:
		if msg.err != nil {
			m.setError(msg.err.Error())
			return m, nil
		}
		m.err = ""
		m.status = "Секрет создан"
		m.setScreen(screenList)
		return m, syncCmd(m.apiClient, m.token, nil)
	case updateDoneMsg:
		if msg.err != nil {
			m.setError(displayError(msg.err))
			return m, nil
		}
		m.err = ""
		m.status = "Секрет обновлен"
		m.setScreen(screenList)
		return m, syncCmd(m.apiClient, m.token, nil)
	case createBlobDoneMsg:
		if msg.err != nil {
			m.setError(msg.err.Error())
			return m, nil
		}
		m.err = ""
		m.status = "Blob-секрет создан"
		m.setScreen(screenList)
		return m, syncCmd(m.apiClient, m.token, nil)
	case downloadBlobDoneMsg:
		if msg.err != nil {
			m.setError(msg.err.Error())
			return m, nil
		}
		m.err = ""
		m.status = "Blob сохранен: " + msg.path
		m.setScreen(screenView)
		return m, nil
	case updateBlobDoneMsg:
		if msg.err != nil {
			m.setError(displayError(msg.err))
			return m, nil
		}
		m.err = ""
		m.status = "Blob-содержимое заменено"
		m.setScreen(screenList)
		return m, syncCmd(m.apiClient, m.token, nil)
	case logoutDoneMsg:
		if msg.err != nil {
			m.setError(msg.err.Error())
			return m, nil
		}
		m.err = ""
		m.status = "Выход выполнен"
		m.token = ""
		m.lastSyncAt = nil
		m.secrets = nil
		m.selected = nil
		refreshSecretsList(&m)
		m.setScreen(screenAuth)
		return m, nil
	case deleteDoneMsg:
		if msg.err != nil {
			if isAPIError(msg.err, http.StatusNotFound, "secret_not_found") {
				removeSecret(&m, msg.id.String())
				m.err = ""
				m.status = "Секрет уже удален"
				m.setScreen(screenList)
				return m, syncCmd(m.apiClient, m.token, nil)
			}
			m.setError(displayError(msg.err))
			return m, nil
		}
		m.err = ""
		m.status = "Секрет удален"
		removeSecret(&m, msg.id.String())
		m.setScreen(screenList)
		return m, syncCmd(m.apiClient, m.token, nil)
	}
	return m.updateCurrent(message)
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		if m.screen == screenList {
			return m, nil
		}
	case "Q":
		if m.screen == screenList {
			return m, tea.Quit
		}
		if m.screen == screenView {
			m.setScreen(screenList)
			return m, nil
		}
	case "esc":
		if m.screen == screenList {
			return m, nil
		}
		if m.screen != screenAuth {
			m.setScreen(screenList)
			return m, nil
		}
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
	case screenConfirmDelete:
		return m.updateConfirmDelete(message)
	case screenConfirmLogout:
		return m.updateConfirmLogout(message)
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
		case "tab":
			m.authInputs[m.authFocus].Blur()
			m.authFocus++
			if m.authFocus > authFieldPassword {
				m.authFocus = authFieldLogin
			}
			m.authInputs[m.authFocus].Focus()
			return m, nil
		case "shift+tab":
			m.register = !m.register
			return m, nil
		case "enter":
			return m, authCmd(m.apiClient, m.authInputs[0].Value(), m.authInputs[1].Value(), m.register)
		}
	}
	var cmd tea.Cmd
	m.authInputs[m.authFocus], cmd = updateInput(m.authInputs[m.authFocus], message)
	return m, cmd
}

func (m Model) updateList(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "r":
			return m, syncCmd(m.apiClient, m.token, m.lastSyncAt)
		case "a":
			resetCreate(&m)
			m.setScreen(screenCreate)
			return m, nil
		case "b":
			resetCreateBlob(&m)
			m.setScreen(screenCreateBlob)
			return m, nil
		case "d":
			selected, ok := m.selectedListItem()
			if !ok {
				return m, nil
			}
			secret := selected.secret
			m.selected = &secret
			m.setScreen(screenConfirmDelete)
			return m, nil
		case "l":
			m.setScreen(screenConfirmLogout)
			return m, nil
		case "enter":
			selected, ok := m.selectedListItem()
			if !ok {
				return m, nil
			}
			secret := selected.secret
			m.selected = &secret
			m.setScreen(screenView)
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.secretsList, cmd = m.secretsList.Update(message)
	return m, cmd
}

func (m Model) updateConfirmDelete(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			if m.selected == nil {
				m.setScreen(screenList)
				return m, nil
			}
			return m, deleteCmd(m.apiClient, m.token, m.selected.ID)
		case "n", "N", "esc":
			m.setScreen(screenList)
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateConfirmLogout(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			return m, logoutCmd(m.store, m.cache)
		case "n", "N", "esc":
			m.setScreen(screenList)
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateView(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "e":
			if m.selected == nil || m.selected.Payload == nil {
				m.setError("выбранный секрет не является structured")
				return m, nil
			}
			resetUpdate(&m)
			m.setScreen(screenUpdate)
			return m, nil
		case "s":
			if m.selected == nil || m.selected.Blob == nil {
				m.setError("выбранный секрет не является blob")
				return m, nil
			}
			resetDownloadBlob(&m)
			m.setScreen(screenDownloadBlob)
			return m, nil
		case "p":
			if m.selected == nil || m.selected.Blob == nil {
				m.setError("выбранный секрет не является blob")
				return m, nil
			}
			resetUpdateBlob(&m)
			m.setScreen(screenUpdateBlob)
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateCreate(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "tab":
			moveStructuredFocus(m.createInputs, &m.createFocus, false)
			return m, nil
		case "shift+tab":
			toggleStructuredType(m.createInputs, &m.createFocus)
			return m, nil
		case "enter":
			input, err := m.createSecretInput()
			if err != nil {
				m.setError(err.Error())
				return m, nil
			}
			return m, createCmd(m.apiClient, m.token, input)
		}
	}
	var cmd tea.Cmd
	m.createInputs[m.createFocus], cmd = updateInput(m.createInputs[m.createFocus], message)
	return m, cmd
}

func (m Model) updateUpdate(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "tab":
			moveStructuredFocus(m.updateInputs, &m.updateFocus, false)
			return m, nil
		case "shift+tab":
			toggleStructuredType(m.updateInputs, &m.updateFocus)
			return m, nil
		case "enter":
			if m.selected == nil || m.selected.Payload == nil {
				m.setError("выбранный секрет не является structured")
				return m, nil
			}
			input, err := m.updateSecretInput()
			if err != nil {
				m.setError(err.Error())
				return m, nil
			}
			return m, updateCmd(m.apiClient, m.token, m.selected.ID, input)
		}
	}
	var cmd tea.Cmd
	m.updateInputs[m.updateFocus], cmd = updateInput(m.updateInputs[m.updateFocus], message)
	return m, cmd
}

func (m Model) updateCreateBlob(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok {
		switch msg.String() {
		case "tab":
			moveBlobFocus(m.blobInputs, &m.blobFocus)
			return m, nil
		case "shift+tab":
			toggleBlobType(m.blobInputs)
			return m, nil
		case "enter":
			input, closeFile, err := m.blobSecretInput()
			if err != nil {
				m.setError(err.Error())
				return m, nil
			}
			return m, createBlobCmd(m.apiClient, m.token, input, closeFile)
		}
	}
	var cmd tea.Cmd
	m.blobInputs[m.blobFocus], cmd = updateInput(m.blobInputs[m.blobFocus], message)
	return m, cmd
}

func (m Model) updateDownloadBlob(message tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok && msg.String() == "enter" {
		if m.selected == nil || m.selected.Blob == nil {
			m.setError("выбранный секрет не является blob")
			return m, nil
		}
		path, file, err := m.downloadBlobOutput()
		if err != nil {
			m.setError(err.Error())
			return m, nil
		}
		return m, downloadBlobCmd(m.apiClient, m.token, m.selected.ID, path, file.Close, file)
	}
	var cmd tea.Cmd
	m.downloadInputs[m.downloadFocus], cmd = updateInput(m.downloadInputs[m.downloadFocus], message)
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
				m.setError("выбранный секрет не является blob")
				return m, nil
			}
			input, closeFile, err := m.updateBlobInput()
			if err != nil {
				m.setError(err.Error())
				return m, nil
			}
			input.ExpectedVersion = m.selected.Version
			return m, updateBlobCmd(m.apiClient, m.token, m.selected.ID, input, closeFile)
		}
	}
	var cmd tea.Cmd
	m.updateBlobInputs[m.updateBlobFocus], cmd = updateInput(m.updateBlobInputs[m.updateBlobFocus], message)
	return m, cmd
}

func displayError(err error) string {
	var apiError api.APIError
	if errors.As(err, &apiError) && apiError.StatusCode == http.StatusConflict {
		return "секрет изменился на сервере, нажми r и повтори"
	}
	return err.Error()
}

func isAPIError(err error, statusCode int, code string) bool {
	var apiError api.APIError
	return errors.As(err, &apiError) && apiError.StatusCode == statusCode && apiError.Code == code
}

func (m *Model) setError(message string) {
	m.err = message
	m.status = ""
}

func (m *Model) setScreen(next screen) {
	if m.screen != next {
		m.err = ""
	}
	m.screen = next
}

func updateInput(input textinput.Model, message tea.Msg) (textinput.Model, tea.Cmd) {
	if msg, ok := message.(tea.KeyMsg); ok && msg.String() == " " {
		message = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	}
	return input.Update(message)
}
