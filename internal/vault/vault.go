package vault

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/braind/braind/internal/gitops"
	"github.com/braind/braind/internal/store"
)

type Vault struct {
	Name  string
	Path  string
	Store *store.Store
	Git   *gitops.GitOps
}

func Open(name, path string) (*Vault, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("vault: abs path: %w", err)
	}

	if err := ensureLayout(absPath); err != nil {
		return nil, fmt.Errorf("vault: ensure layout: %w", err)
	}

	st, err := store.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("vault: open store: %w", err)
	}

	g, err := gitops.Init(absPath, "braind", "braind@localhost")
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("vault: init git: %w", err)
	}

	return &Vault{
		Name:  name,
		Path:  absPath,
		Store: st,
		Git:   g,
	}, nil
}

func ensureLayout(path string) error {
	dirs := []string{
		filepath.Join(path, "journals"),
		filepath.Join(path, "pages"),
		filepath.Join(path, ".braind"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("vault: mkdir %s: %w", dir, err)
		}
	}

	files := []string{
		filepath.Join(path, "changelog.org"),
	}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			f, err := os.Create(file)
			if err != nil {
				return fmt.Errorf("vault: create %s: %w", file, err)
			}
			f.Close()
		}
	}

	return nil
}

func (v *Vault) Close() error {
	if v.Store != nil {
		v.Store.Close()
	}
	return nil
}

func (v *Vault) JournalPath(date string) string {
	return filepath.Join(v.Path, "journals", date+".md")
}

func (v *Vault) PagePath(name string) string {
	return filepath.Join(v.Path, "pages", name+".md")
}
