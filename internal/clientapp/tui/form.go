package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/igor/gophkeeper/internal/clientapp/api"
)

func (m *Model) resetCreate() {
	for index := range m.createInputs {
		m.createInputs[index].SetValue("")
		m.createInputs[index].Blur()
	}
	m.createInputs[0].SetValue(string(api.SecretTypeCredentials))
	m.createFocus = 0
	m.createInputs[0].Focus()
}

func (m Model) createSecretInput() (api.SecretInput, error) {
	secretType := api.SecretType(strings.TrimSpace(m.createInputs[createFieldType].Value()))
	if secretType != api.SecretTypeCredentials && secretType != api.SecretTypeCard {
		return api.SecretInput{}, fmt.Errorf("type должен быть credentials или card")
	}
	metadata := make(map[string]interface{})
	if value := strings.TrimSpace(m.createInputs[createFieldMetadata].Value()); value != "" {
		if err := json.Unmarshal([]byte(value), &metadata); err != nil {
			return api.SecretInput{}, fmt.Errorf("metadata должен быть JSON object: %w", err)
		}
	}
	payload := map[string]interface{}{}
	if secretType == api.SecretTypeCredentials {
		payload["login"] = m.createInputs[createFieldFirst].Value()
		payload["password"] = m.createInputs[createFieldSecond].Value()
	} else {
		payload["number"] = m.createInputs[createFieldFirst].Value()
		payload["holder"] = m.createInputs[createFieldSecond].Value()
		payload["expires_at"] = m.createInputs[createFieldThird].Value()
		payload["cvv"] = m.createInputs[createFieldFourth].Value()
	}
	return api.SecretInput{
		Type:     secretType,
		Name:     m.createInputs[createFieldName].Value(),
		Metadata: metadata,
		Payload:  payload,
	}, nil
}

func newInput(placeholder string) textinput.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = 256
	input.Width = 60
	return input
}

func newPasswordInput(placeholder string) textinput.Model {
	input := newInput(placeholder)
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '*'
	return input
}

func formatMap(value map[string]interface{}) string {
	if value == nil {
		return "{}"
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(payload)
}
