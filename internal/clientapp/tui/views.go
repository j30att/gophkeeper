package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
)

// View возвращает строковое представление TUI.
func (m Model) View() string {
	var body string
	switch m.screen {
	case screenAuth:
		body = m.viewAuth()
	case screenList:
		body = m.viewList()
	case screenView:
		body = m.viewSecret()
	case screenConfirmDelete:
		body = m.viewConfirmDelete()
	case screenConfirmLogout:
		body = m.viewConfirmLogout()
	case screenCreate:
		body = m.viewCreate()
	case screenUpdate:
		body = m.viewUpdate()
	case screenCreateBlob:
		body = m.viewCreateBlob()
	case screenDownloadBlob:
		body = m.viewDownloadBlob()
	case screenUpdateBlob:
		body = m.viewUpdateBlob()
	}
	if m.err != "" {
		body += "\n" + errorStyle.Render(m.err)
	}
	if m.status != "" {
		body += "\n" + helpStyle.Render(m.status)
	}
	return body
}

func (m Model) viewAuth() string {
	mode := "login"
	if m.register {
		mode = "register"
	}
	return strings.Join(
		[]string{
			titleStyle.Render("GophKeeper"),
			m.authInputs[0].View(),
			m.authInputs[1].View(),
			fmt.Sprintf("mode: %s  [shift+tab - переключить]", mode),
			helpStyle.Render("shift+tab - login/register, enter - выполнить, tab - следующее поле, shift+q - выход"),
		},
		"\n",
	)
}

func (m Model) viewList() string {
	return titleStyle.Render("GophKeeper secrets") + "\n" +
		m.secretsList.View() + "\n" +
		helpStyle.Render("enter - открыть, a - создать JSON, b - создать blob, d - удалить, r - sync, l - logout, shift+q - выход")
}

func (m Model) viewSecret() string {
	if m.selected == nil {
		return "Секрет не выбран"
	}
	secret := *m.selected
	lines := []string{
		titleStyle.Render(secret.Name),
		fmt.Sprintf("type: %s", secret.Type),
		fmt.Sprintf("version: %d", secret.Version),
		fmt.Sprintf("updated_at: %s", secret.UpdatedAt.Format(time.RFC3339)),
		"description:",
		formatDescriptionMetadata(secret.Metadata),
	}
	if secret.Payload != nil {
		lines = append(lines, "payload:", formatMap(secret.Payload))
		lines = append(lines, helpStyle.Render("e - редактировать"))
	}
	if secret.Blob != nil {
		lines = append(lines, "blob:", fmt.Sprintf("%s, %d bytes", secret.Blob.OriginalName, secret.Blob.Size))
		lines = append(lines, helpStyle.Render("s - скачать, p - заменить файл"))
	}
	lines = append(lines, helpStyle.Render("esc/shift+q - назад"))
	return strings.Join(lines, "\n")
}

func (m Model) viewCreate() string {
	lines := []string{titleStyle.Render("Создать structured secret")}
	lines = append(lines, m.viewStructuredInputs(m.createInputs)...)
	lines = append(lines, helpStyle.Render("shift+tab - переключить credentials/card, tab - следующее поле, enter - создать, esc - назад"))
	return strings.Join(lines, "\n")
}

func (m Model) viewUpdate() string {
	lines := []string{titleStyle.Render("Редактировать structured secret")}
	if m.selected != nil {
		lines = append(lines, fmt.Sprintf("expected_version: %d", m.selected.Version))
	}
	lines = append(lines, m.viewStructuredInputs(m.updateInputs)...)
	lines = append(lines, helpStyle.Render("shift+tab - переключить credentials/card, tab - следующее поле, enter - сохранить, esc - назад"))
	return strings.Join(lines, "\n")
}

func (m Model) viewConfirmDelete() string {
	name := ""
	if m.selected != nil {
		name = m.selected.Name
	}
	return strings.Join(
		[]string{
			titleStyle.Render("Удалить secret?"),
			"name: " + name,
			helpStyle.Render("y - удалить, n/esc - отмена"),
		},
		"\n",
	)
}

func (m Model) viewConfirmLogout() string {
	return strings.Join(
		[]string{
			titleStyle.Render("Выйти из клиента?"),
			"Будут удалены локальные session.json и secrets.json.",
			helpStyle.Render("y - logout, n/esc - отмена"),
		},
		"\n",
	)
}

func (m Model) viewStructuredInputs(inputs []textinput.Model) []string {
	labels := structuredInputLabels(inputs)
	fields := visibleStructuredFields(inputs)
	lines := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == createFieldType {
			lines = append(lines, labels[field]+": "+inputs[field].Value()+"  [shift+tab - сменить]")
			continue
		}
		input := inputs[field]
		input.Placeholder = labels[field]
		lines = append(lines, labels[field]+": "+input.View())
	}
	return lines
}

func (m Model) viewCreateBlob() string {
	lines := []string{titleStyle.Render("Создать blob secret")}
	lines = append(lines, "type: "+m.blobInputs[blobFieldType].Value()+"  [shift+tab - сменить]")
	for _, field := range visibleBlobFields() {
		input := m.blobInputs[field]
		lines = append(lines, input.View())
	}
	lines = append(lines, helpStyle.Render("shift+tab - переключить text/binary, tab - следующее поле, enter - загрузить, esc - назад"))
	return strings.Join(lines, "\n")
}

func (m Model) viewDownloadBlob() string {
	lines := []string{titleStyle.Render("Скачать blob secret")}
	if m.selected != nil && m.selected.Blob != nil {
		lines = append(lines, "file: "+m.selected.Blob.OriginalName)
	}
	for _, input := range m.downloadInputs {
		lines = append(lines, input.View())
	}
	lines = append(lines, helpStyle.Render("пустой path сохранит в original filename; enter - скачать, esc - назад"))
	return strings.Join(lines, "\n")
}

func (m Model) viewUpdateBlob() string {
	lines := []string{titleStyle.Render("Заменить blob content")}
	if m.selected != nil {
		lines = append(lines, fmt.Sprintf("expected_version: %d", m.selected.Version))
	}
	for _, input := range m.updateBlobInputs {
		lines = append(lines, input.View())
	}
	lines = append(lines, helpStyle.Render("enter - заменить, esc - назад"))
	return strings.Join(lines, "\n")
}
