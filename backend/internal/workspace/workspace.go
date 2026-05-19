package workspace

import (
	"os"
	"path/filepath"
)

// Workspace represents a company or session workspace.
type Workspace struct {
	BaseDir string
}

// NewWorkspace creates a workspace manager.
func NewWorkspace(baseDir string) *Workspace {
	return &Workspace{BaseDir: baseDir}
}

// EnsureDir creates directory if not exists.
func (w *Workspace) EnsureDir(subPath string) error {
	dir := filepath.Join(w.BaseDir, subPath)
	return os.MkdirAll(dir, 0755)
}

// WriteFile writes content to a file in workspace.
func (w *Workspace) WriteFile(subPath, content string) error {
	path := filepath.Join(w.BaseDir, subPath)
	if err := w.EnsureDir(filepath.Dir(subPath)); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// ReadFile reads content from a file in workspace.
func (w *Workspace) ReadFile(subPath string) (string, error) {
	path := filepath.Join(w.BaseDir, subPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ListFiles returns all files in a subdirectory.
func (w *Workspace) ListFiles(subPath string) ([]string, error) {
	dir := filepath.Join(w.BaseDir, subPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

// Exists checks if a file exists.
func (w *Workspace) Exists(subPath string) bool {
	path := filepath.Join(w.BaseDir, subPath)
	_, err := os.Stat(path)
	return err == nil
}