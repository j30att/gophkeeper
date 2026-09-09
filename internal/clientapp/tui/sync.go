package tui

import (
	"sort"

	"github.com/charmbracelet/bubbles/list"

	"github.com/igor/gophkeeper/internal/clientapp/api"
)

func applySync(m *Model, response api.SyncResponse) {
	byID := make(map[string]api.Secret, len(m.secrets))
	for _, secret := range m.secrets {
		byID[secret.ID.String()] = secret
	}
	for _, secret := range response.Secrets {
		byID[secret.ID.String()] = secret
	}
	for _, deleted := range response.Deleted {
		delete(byID, deleted.ID.String())
	}
	m.secrets = make([]api.Secret, 0, len(byID))
	for _, secret := range byID {
		m.secrets = append(m.secrets, secret)
	}
	sort.Slice(m.secrets, func(i int, j int) bool {
		return m.secrets[i].UpdatedAt.After(m.secrets[j].UpdatedAt)
	})
	refreshSecretsList(m)
}

func refreshSecretsList(m *Model) {
	items := make([]list.Item, 0, len(m.secrets))
	for _, secret := range m.secrets {
		items = append(items, secretListItem{secret: secret})
	}
	m.secretsList.SetItems(items)
}

func removeSecret(m *Model, id string) {
	secrets := make([]api.Secret, 0, len(m.secrets))
	for _, secret := range m.secrets {
		if secret.ID.String() != id {
			secrets = append(secrets, secret)
		}
	}
	m.secrets = secrets
	if m.selected != nil && m.selected.ID.String() == id {
		m.selected = nil
	}
	refreshSecretsList(m)
}
