package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// ProjectManager gestiona la resolución de bases de datos basadas en el directorio de trabajo.
type ProjectManager struct {
	baseDir string
}

// NewProjectManager crea un nuevo gestor de proyectos.
func NewProjectManager(baseDir string) (*ProjectManager, error) {
	// Asegurar que el directorio base de Vectos existe
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &ProjectManager{baseDir: baseDir}, nil
}

// BaseDir devuelve el directorio base donde Vectos guarda índices por proyecto.
func (pm *ProjectManager) BaseDir() string {
	return pm.baseDir
}

// GetDatabasePath para el directorio actual determina la ruta de la DB para el proyecto actual.
func (pm *ProjectManager) GetDatabasePath(currentDir string) (string, error) {
	projectName, err := projectNameFromDir(currentDir)
	if err != nil {
		return "", err
	}

	return pm.GetDatabasePathForName(projectName)
}

// GetDatabasePathForName determina la ruta de la DB para un nombre lógico de proyecto.
func (pm *ProjectManager) GetDatabasePathForName(projectName string) (string, error) {
	projectKey := normalizeProjectName(projectName)
	dbName := fmt.Sprintf("%s.db", projectKey)
	return filepath.Join(pm.baseDir, projectKey, dbName), nil
}

// EnsureProjectDir crea el directorio específico para el proyecto si no existe.
func (pm *ProjectManager) EnsureProjectDir(currentDir string) (string, error) {
	projectName, err := projectNameFromDir(currentDir)
	if err != nil {
		return "", err
	}

	return pm.EnsureProjectDirForName(projectName)
}

// EnsureProjectDirForName crea el directorio específico para un nombre lógico de proyecto.
func (pm *ProjectManager) EnsureProjectDirForName(projectName string) (string, error) {
	projectPath := filepath.Join(pm.baseDir, normalizeProjectName(projectName))

	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %w", err)
	}

	return projectPath, nil
}

func projectNameFromDir(currentDir string) (string, error) {
	absDir, err := filepath.Abs(currentDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	projectName := filepath.Base(absDir)
	if projectName == "/" || projectName == "." || projectName == "" {
		projectName = "default"
	}

	return projectName, nil
}

func normalizeProjectName(projectName string) string {
	trimmed := strings.TrimSpace(projectName)
	if trimmed == "" {
		return "default"
	}

	var b strings.Builder
	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}

	normalized := strings.Trim(b.String(), "-.")
	if normalized == "" {
		return "default"
	}

	return normalized
}
