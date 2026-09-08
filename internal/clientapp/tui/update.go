package tui

import tea "github.com/charmbracelet/bubbletea"

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
			return m, syncCmd(m.apiClient, m.token, m.lastSyncAt)
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
		m.applySync(msg.response)
		return m, nil
	case createDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = "Секрет создан"
		m.screen = screenList
		return m, syncCmd(m.apiClient, m.token, nil)
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
	case screenCreate:
		return m.updateCreate(message)
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
			m.resetCreate()
			m.screen = screenCreate
			return m, nil
		case "d":
			selected, ok := m.selectedListItem()
			if !ok {
				return m, nil
			}
			return m, deleteCmd(m.apiClient, m.token, selected.secret.ID)
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
