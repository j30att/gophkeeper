package tui

import (
	"fmt"
	"time"

	"github.com/igor/gophkeeper/internal/clientapp/api"
)

type secretListItem struct {
	secret api.Secret
}

// Title возвращает название секрета для элемента списка.
func (i secretListItem) Title() string {
	return i.secret.Name
}

// Description возвращает описание элемента списка секретов.
func (i secretListItem) Description() string {
	return fmt.Sprintf("%s  v%d  %s", i.secret.Type, i.secret.Version, i.secret.UpdatedAt.Format(time.RFC3339))
}

// FilterValue возвращает значение элемента для фильтрации списка.
func (i secretListItem) FilterValue() string {
	return i.secret.Name
}

func (m Model) selectedListItem() (secretListItem, bool) {
	item, ok := m.secretsList.SelectedItem().(secretListItem)
	return item, ok
}
