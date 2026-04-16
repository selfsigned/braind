package gitops

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

type GitOps struct {
	repo  *git.Repository
	path  string
	name  string
	email string
}

func Init(path, name, email string) (*GitOps, error) {
	r, err := git.PlainInit(path, false)
	if err != nil {
		r, err = git.PlainOpen(path)
		if err != nil {
			return nil, fmt.Errorf("gitops: open: %w", err)
		}
	}

	cfg, _ := r.Config()
	cfg.User.Name = name
	cfg.User.Email = email
	r.SetConfig(cfg)

	return &GitOps{
		repo:  r,
		path:  path,
		name:  name,
		email: email,
	}, nil
}

func (g *GitOps) Commit(message string, changes []FileChange) (string, error) {
	wt, err := g.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("gitops: worktree: %w", err)
	}

	for _, c := range changes {
		fullPath := filepath.Join(g.path, c.Path)
		if c.Action == "delete" {
			_, err := wt.Remove(c.Path)
			if err != nil {
				return "", fmt.Errorf("gitops: remove %s: %w", c.Path, err)
			}
		} else {
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				dir := filepath.Dir(fullPath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					return "", fmt.Errorf("gitops: mkdir %s: %w", dir, err)
				}
			}
			_, err := wt.Add(c.Path)
			if err != nil {
				return "", fmt.Errorf("gitops: add %s: %w", c.Path, err)
			}
		}
	}

	hash, err := wt.Commit(message, &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  g.name,
			Email: g.email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("gitops: commit: %w", err)
	}

	return hash.String(), nil
}

func (g *GitOps) Revert(ctx interface{}, commitHash string) (string, error) {
	if commitHash == "" {
		ref, err := g.repo.Head()
		if err != nil {
			return "", fmt.Errorf("gitops: head: %w", err)
		}
		commitHash = ref.Hash().String()
	}

	hash := plumbing.NewHash(commitHash)
	commit, err := g.repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("gitops: commit object: %w", err)
	}

	if len(commit.ParentHashes) == 0 {
		return "", fmt.Errorf("gitops: revert: cannot revert initial commit")
	}

	parentCommit, err := g.repo.CommitObject(commit.ParentHashes[0])
	if err != nil {
		return "", fmt.Errorf("gitops: parent object: %w", err)
	}

	currentTree, err := commit.Tree()
	if err != nil {
		return "", fmt.Errorf("gitops: current tree: %w", err)
	}

	parentTree, err := parentCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("gitops: parent tree: %w", err)
	}

	changes, err := currentTree.Diff(parentTree)
	if err != nil {
		return "", fmt.Errorf("gitops: diff: %w", err)
	}

	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			continue
		}

		var fromFile, toFile *object.File
		fromFile, toFile, err = change.Files()
		if err != nil {
			continue
		}

		var content string
		var fullPath string

		switch action {
		case merkletrie.Delete:
			if fromFile != nil {
				content, _ = fromFile.Contents()
				fullPath = filepath.Join(g.path, change.From.Name)
			}
		case merkletrie.Insert, merkletrie.Modify:
			if toFile != nil {
				content, _ = toFile.Contents()
				fullPath = filepath.Join(g.path, change.To.Name)
			}
		}

		if content != "" && fullPath != "" {
			os.MkdirAll(filepath.Dir(fullPath), 0755)
			os.WriteFile(fullPath, []byte(content), 0644)
		}
	}

	wt, err := g.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("gitops: worktree: %w", err)
	}

	_, err = wt.Add(".")
	if err != nil {
		return "", fmt.Errorf("gitops: add: %w", err)
	}

	revertMsg := fmt.Sprintf("Revert: %s", commit.Message)

	newHash, err := wt.Commit(revertMsg, &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  g.name,
			Email: g.email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("gitops: revert commit: %w", err)
	}

	return newHash.String(), nil
}

func (g *GitOps) LastCommit() (string, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return "", fmt.Errorf("gitops: head: %w", err)
	}

	return ref.Hash().String(), nil
}

func (g *GitOps) Log() ([]string, error) {
	ref, err := g.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("gitops: head: %w", err)
	}

	c, err := g.repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("gitops: commit object: %w", err)
	}

	var commits []string
	for c != nil {
		commits = append(commits, c.Message)
		c, err = c.Parent(0)
		if err != nil {
			break
		}
	}

	return commits, nil
}

type FileChange struct {
	Path   string
	Action string
	Diff   string
}
