package tui

import (
	"fmt"
	"strings"
	"time"
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
			fmt.Sprintf("mode: %s  [space - переключить]", mode),
			helpStyle.Render("enter - выполнить, tab - следующее поле, q - выход"),
		},
		"\n",
	)
}

func (m Model) viewList() string {
	return titleStyle.Render("GophKeeper secrets") + "\n" +
		m.secretsList.View() + "\n" +
		helpStyle.Render("enter - открыть, a - создать JSON, b - создать blob, d - удалить, r - sync, l - logout, q - выход")
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
		"metadata:",
		formatMap(secret.Metadata),
	}
	if secret.Payload != nil {
		lines = append(lines, "payload:", formatMap(secret.Payload))
		lines = append(lines, helpStyle.Render("e - редактировать"))
	}
	if secret.Blob != nil {
		lines = append(lines, "blob:", fmt.Sprintf("%s, %d bytes", secret.Blob.OriginalName, secret.Blob.Size))
		lines = append(lines, helpStyle.Render("s - скачать, p - заменить файл"))
	}
	lines = append(lines, helpStyle.Render("esc/q - назад"))
	return strings.Join(lines, "\n")
}

func (m Model) viewCreate() string {
	lines := []string{titleStyle.Render("Создать structured secret")}
	for _, input := range m.createInputs {
		lines = append(lines, input.View())
	}
	lines = append(lines, helpStyle.Render("type: credentials или card; enter - создать, esc - назад"))
	return strings.Join(lines, "\n")
}

func (m Model) viewUpdate() string {
	lines := []string{titleStyle.Render("Редактировать structured secret")}
	if m.selected != nil {
		lines = append(lines, fmt.Sprintf("expected_version: %d", m.selected.Version))
	}
	for _, input := range m.updateInputs {
		lines = append(lines, input.View())
	}
	lines = append(lines, helpStyle.Render("type: credentials или card; enter - сохранить, esc - назад"))
	return strings.Join(lines, "\n")
}

func (m Model) viewCreateBlob() string {
	lines := []string{titleStyle.Render("Создать blob secret")}
	for _, input := range m.blobInputs {
		lines = append(lines, input.View())
	}
	lines = append(lines, helpStyle.Render("type: text или binary; enter - загрузить, esc - назад"))
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
