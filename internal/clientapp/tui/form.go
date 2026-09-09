package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/igor/gophkeeper/internal/clientapp/api"
)

func resetCreate(m *Model) {
	for index := range m.createInputs {
		m.createInputs[index].SetValue("")
		m.createInputs[index].Blur()
	}
	m.createInputs[0].SetValue(string(api.SecretTypeCredentials))
	m.createFocus = 0
	m.createInputs[0].Focus()
}

func resetCreateBlob(m *Model) {
	for index := range m.blobInputs {
		m.blobInputs[index].SetValue("")
		m.blobInputs[index].Blur()
	}
	m.blobInputs[blobFieldType].SetValue(string(api.SecretTypeBinary))
	m.blobFocus = 0
	m.blobInputs[0].Focus()
}

func resetUpdate(m *Model) {
	for index := range m.updateInputs {
		m.updateInputs[index].SetValue("")
		m.updateInputs[index].Blur()
	}
	m.updateFocus = 0
	if m.selected != nil {
		m.fillStructuredInputs(m.updateInputs, *m.selected)
	}
	m.updateInputs[0].Focus()
}

func resetDownloadBlob(m *Model) {
	for index := range m.downloadInputs {
		m.downloadInputs[index].SetValue("")
		m.downloadInputs[index].Blur()
	}
	if m.selected != nil && m.selected.Blob != nil {
		m.downloadInputs[downloadFieldPath].SetValue(m.selected.Blob.OriginalName)
	}
	m.downloadFocus = 0
	m.downloadInputs[0].Focus()
}

func resetUpdateBlob(m *Model) {
	for index := range m.updateBlobInputs {
		m.updateBlobInputs[index].SetValue("")
		m.updateBlobInputs[index].Blur()
	}
	m.updateBlobFocus = 0
	m.updateBlobInputs[0].Focus()
}

func (m Model) createSecretInput() (api.SecretInput, error) {
	return structuredSecretInput(m.createInputs, 0)
}

func (m Model) updateSecretInput() (api.SecretInput, error) {
	expectedVersion := 0
	if m.selected != nil {
		expectedVersion = m.selected.Version
	}
	return structuredSecretInput(m.updateInputs, expectedVersion)
}

func structuredSecretInput(inputs []textinput.Model, expectedVersion int) (api.SecretInput, error) {
	secretType := api.SecretType(strings.TrimSpace(inputs[createFieldType].Value()))
	if secretType != api.SecretTypeCredentials && secretType != api.SecretTypeCard {
		return api.SecretInput{}, fmt.Errorf("type должен быть credentials или card")
	}
	metadata := make(map[string]interface{})
	if value := strings.TrimSpace(inputs[createFieldMetadata].Value()); value != "" {
		if err := json.Unmarshal([]byte(value), &metadata); err != nil {
			return api.SecretInput{}, fmt.Errorf("metadata должен быть JSON object: %w", err)
		}
	}
	payload := map[string]interface{}{}
	if secretType == api.SecretTypeCredentials {
		payload["login"] = inputs[createFieldFirst].Value()
		payload["password"] = inputs[createFieldSecond].Value()
	} else {
		payload["number"] = inputs[createFieldFirst].Value()
		payload["holder"] = inputs[createFieldSecond].Value()
		payload["expires_at"] = inputs[createFieldThird].Value()
		payload["cvv"] = inputs[createFieldFourth].Value()
	}
	return api.SecretInput{
		Type:            secretType,
		Name:            inputs[createFieldName].Value(),
		Metadata:        metadata,
		Payload:         payload,
		ExpectedVersion: expectedVersion,
	}, nil
}

func (m Model) fillStructuredInputs(inputs []textinput.Model, secret api.Secret) {
	inputs[createFieldType].SetValue(string(secret.Type))
	inputs[createFieldName].SetValue(secret.Name)
	inputs[createFieldMetadata].SetValue(formatMap(secret.Metadata))
	if secret.Type == api.SecretTypeCredentials {
		inputs[createFieldFirst].SetValue(stringValue(secret.Payload, "login"))
		inputs[createFieldSecond].SetValue(stringValue(secret.Payload, "password"))
		return
	}
	inputs[createFieldFirst].SetValue(stringValue(secret.Payload, "number"))
	inputs[createFieldSecond].SetValue(stringValue(secret.Payload, "holder"))
	inputs[createFieldThird].SetValue(stringValue(secret.Payload, "expires_at"))
	inputs[createFieldFourth].SetValue(stringValue(secret.Payload, "cvv"))
}

func stringValue(values map[string]interface{}, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func (m Model) blobSecretInput() (api.BlobSecretInput, func() error, error) {
	secretType := api.SecretType(strings.TrimSpace(m.blobInputs[blobFieldType].Value()))
	if secretType != api.SecretTypeText && secretType != api.SecretTypeBinary {
		return api.BlobSecretInput{}, nil, fmt.Errorf("type должен быть text или binary")
	}
	path := strings.TrimSpace(m.blobInputs[blobFieldPath].Value())
	if path == "" {
		return api.BlobSecretInput{}, nil, fmt.Errorf("file path обязателен")
	}
	file, err := os.Open(path)
	if err != nil {
		return api.BlobSecretInput{}, nil, fmt.Errorf("open file: %w", err)
	}
	name := strings.TrimSpace(m.blobInputs[blobFieldName].Value())
	if name == "" {
		name = filepath.Base(path)
	}
	return api.BlobSecretInput{
		Type:         secretType,
		Name:         name,
		Metadata:     strings.TrimSpace(m.blobInputs[blobFieldMetadata].Value()),
		OriginalName: filepath.Base(path),
		ContentType:  strings.TrimSpace(m.blobInputs[blobFieldContentType].Value()),
		Content:      file,
	}, file.Close, nil
}

func (m Model) downloadBlobOutput() (string, *os.File, error) {
	path := strings.TrimSpace(m.downloadInputs[downloadFieldPath].Value())
	if path == "" && m.selected != nil && m.selected.Blob != nil {
		path = m.selected.Blob.OriginalName
	}
	if path == "" {
		return "", nil, fmt.Errorf("save path обязателен")
	}
	file, err := os.Create(path)
	if err != nil {
		return "", nil, fmt.Errorf("create file: %w", err)
	}
	return path, file, nil
}

func (m Model) updateBlobInput() (api.BlobContentInput, func() error, error) {
	path := strings.TrimSpace(m.updateBlobInputs[updateBlobFieldPath].Value())
	if path == "" {
		return api.BlobContentInput{}, nil, fmt.Errorf("file path обязателен")
	}
	file, err := os.Open(path)
	if err != nil {
		return api.BlobContentInput{}, nil, fmt.Errorf("open file: %w", err)
	}
	return api.BlobContentInput{
		OriginalName: filepath.Base(path),
		ContentType:  strings.TrimSpace(m.updateBlobInputs[updateBlobFieldContentType].Value()),
		Content:      file,
	}, file.Close, nil
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
