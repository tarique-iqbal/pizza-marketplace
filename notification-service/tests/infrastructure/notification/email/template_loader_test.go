package email_test

import (
	"os"
	"path/filepath"
	"testing"

	"notification-service/internal/infrastructure/notification/email"

	"github.com/stretchr/testify/assert"
)

func TestTextTemplateLoader_Render(t *testing.T) {
	tmpDir := t.TempDir()
	templateName := "test_template.html"
	templatePath := filepath.Join(tmpDir, templateName)

	content := `Hello, {{.name}}!`
	err := os.WriteFile(templatePath, []byte(content), 0644)
	assert.NoError(t, err)

	loader := email.NewTextTemplateLoader(tmpDir)

	result, err := loader.Render(templateName, map[string]string{
		"name": "Alice",
	})

	assert.NoError(t, err)
	assert.Equal(t, "Hello, Alice!", result)
}

func TestTextTemplateLoader_Render_DoesNotEscapeSpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	templateName := "test_template.html"
	templatePath := filepath.Join(tmpDir, templateName)

	content := `Restaurant "{{.name}}" is ready.`
	err := os.WriteFile(templatePath, []byte(content), 0644)
	assert.NoError(t, err)

	loader := email.NewTextTemplateLoader(tmpDir)

	result, err := loader.Render(templateName, map[string]string{
		"name": "Domino's Pizza & Grill",
	})

	assert.NoError(t, err)
	assert.Equal(t, `Restaurant "Domino's Pizza & Grill" is ready.`, result)
}

func TestTextTemplateLoader_Render_FileNotFound(t *testing.T) {
	loader := email.NewTextTemplateLoader("nonexistent")
	_, err := loader.Render("nope.html", nil)
	assert.Error(t, err)
}
