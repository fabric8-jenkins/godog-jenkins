package github

import (
	"os"
	"path/filepath"

	"github.com/google/go-github/github"
	"fmt"
	"strings"
	"os/exec"
)

type GitCommander struct {
	Dir string
	UseHttps bool
}

func CreateGitCommander() *GitCommander {
	dir := os.Getenv("WORK_DIR")
	if len(dir) == 0 {
		dir = "work"
	}
	return &GitCommander{
		Dir: dir,
	}
}

// Clone performs a clone in the directory for the repository
func (g *GitCommander) Clone(repo *github.Repository) (string, error) {
	owner := repo.Owner
	if owner == nil {
		return "", fmt.Errorf("No owner for the repository %v", repo)
	}
	ownerName := owner.Login
	if ownerName == nil {
		return "", fmt.Errorf("No owner login for the repository %v", repo)
	}
	repoName := repo.Name
	if repoName == nil {
		return "", fmt.Errorf("No repo name for the repository %v", repo)
	}
	outDir := filepath.Join(g.Dir, *ownerName, *repoName)

	cloneUrl, err := g.CloneURL(repo)
	if err != nil {
		return outDir, err
	}
	runDir := filepath.Join(g.Dir, *ownerName)
	err = os.MkdirAll(runDir, 0770)
	if err != nil {
		return outDir, err
	}

	err = runCommand(runDir, "git", "clone", cloneUrl)
	return outDir, err
}

// CloneURL returns the URL used to clone this repository
func (g *GitCommander) CloneURL(repo *github.Repository) (string, error) {
	cloneUrl := repo.SSHURL
	if g.UseHttps {
		cloneUrl = repo.CloneURL
		if cloneUrl == nil {
			return "", fmt.Errorf("Git repository does not have a clone URL: %v", repo)
		}
	} else {
		if cloneUrl == nil {
			return "", fmt.Errorf("Git repository does not have a SSH URL: %v", repo)
		}
	}
	return *cloneUrl, nil
}

// runCommand runs the given command in the directory
func runCommand(dir string, prog string, args ...string) error {
	cmd := exec.Command(prog, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		text := prog + " " + strings.Join(args, " ")
		return fmt.Errorf("Failed to run command %s in dir %s due to error %v", text, dir, err)
	}
	return nil
}