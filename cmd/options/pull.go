package options

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	flaggy "github.com/vedadiyan/flaggy/pkg"
	"github.com/vedadiyan/protov/internal/system/install"
)

var (
	ErrNoRepo          = errors.New("repository URI is required")
	ErrInvalidRepo     = errors.New("invalid repository URI")
	ErrProtoPathNotSet = errors.New("proto path not configured")
	ErrGitNotFound     = errors.New("git is not installed or not in PATH")
	ErrCloneFailed     = errors.New("git clone failed")
	ErrDirectoryCreate = errors.New("failed to create directory")
)

type (
	Pull struct {
		Proto    Proto    `long:"proto" help:"pulls protobuffer dependencies"`
		Template Template `long:"template" help:"pulls template dependencies"`
		Help     bool     `long:"help" help:"shows help"`
	}

	Proto struct {
		Repo string `long:"--repo" help:"link to the repository"`
		Help bool   `long:"help" help:"shows help"`
	}
	Template struct {
		Repo string `long:"--repo" help:"link to the template repository"`
		Help bool   `long:"help" help:"shows help"`
	}
)

func (p *Pull) Run() error {
	flaggy.PrintHelp()
	return nil
}

func (p *Proto) Run() error {
	if p.Help {
		flaggy.PrintHelp()
		return nil
	}

	if err := p.validate(); err != nil {
		flaggy.PrintHelp()
		return err
	}

	if err := p.checkPrerequisites(); err != nil {
		return fmt.Errorf("prerequisite check failed: %w", err)
	}

	return p.pullRepository()
}

func (p *Proto) validate() error {
	if len(p.Repo) == 0 {
		return ErrNoRepo
	}

	if err := validateRepoURI(p.Repo); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRepo, err)
	}

	return nil
}

func (p *Proto) checkPrerequisites() error {
	if err := CheckTool("git"); err != nil {
		return ErrGitNotFound
	}
	return nil
}

func (p *Proto) pullRepository() error {
	protoPath, err := install.ProtoPath()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProtoPathNotSet, err)
	}

	if protoPath == "" {
		return ErrProtoPathNotSet
	}

	includeDir := filepath.Join(protoPath, "include")

	if err := EnsureDirectory(includeDir, 0755); err != nil {
		return fmt.Errorf("%w: %v", ErrDirectoryCreate, err)
	}

	if err := ValidateOutputPath(includeDir); err != nil {
		return fmt.Errorf("include directory not writable: %w", err)
	}

	if err := p.cloneRepo(p.Repo, includeDir); err != nil {
		return fmt.Errorf("%w: %v", ErrCloneFailed, err)
	}

	return nil
}

func (p *Proto) cloneRepo(repoURI, targetDir string) error {
	args := []string{"clone", repoURI}

	args = append(args, "--depth", "1") // Shallow clone for efficiency
	args = append(args, "--single-branch")

	if err := Exec("git", targetDir, args...); err != nil {
		return err
	}

	return nil
}

func (t *Template) Run() error {
	if t.Help {
		flaggy.PrintHelp()
		return nil
	}

	if err := t.validate(); err != nil {
		flaggy.PrintHelp()
		return err
	}

	if err := t.checkPrerequisites(); err != nil {
		return fmt.Errorf("prerequisite check failed: %w", err)
	}

	return t.pullRepository()
}

func (t *Template) validate() error {
	if len(t.Repo) == 0 {
		return ErrNoRepo
	}

	if err := validateRepoURI(t.Repo); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRepo, err)
	}

	return nil
}

func (t *Template) checkPrerequisites() error {
	if err := CheckTool("git"); err != nil {
		return ErrGitNotFound
	}
	return nil
}

func (t *Template) pullRepository() error {
	protoPath, err := install.ProtoPath()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProtoPathNotSet, err)
	}

	if protoPath == "" {
		return ErrProtoPathNotSet
	}

	templatesDir := filepath.Join(protoPath, "templates")

	if err := EnsureDirectory(templatesDir, 0755); err != nil {
		return fmt.Errorf("%w: %v", ErrDirectoryCreate, err)
	}

	if err := ValidateOutputPath(templatesDir); err != nil {
		return fmt.Errorf("templates directory not writable: %w", err)
	}

	if err := t.cloneRepo(t.Repo, templatesDir); err != nil {
		return fmt.Errorf("%w: %v", ErrCloneFailed, err)
	}

	return nil
}

func (t *Template) cloneRepo(repoURI, targetDir string) error {
	args := []string{"clone", repoURI}

	args = append(args, "--depth", "1")
	args = append(args, "--single-branch")

	if err := Exec("git", targetDir, args...); err != nil {
		return err
	}

	return nil
}

func validateRepoURI(repo string) error {
	if repo == "" {
		return errors.New("empty repository URI")
	}

	repo = strings.TrimSpace(repo)

	if strings.Contains(repo, "\x00") {
		return errors.New("repository URI contains null byte")
	}

	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		parsedURL, err := url.Parse(repo)
		if err != nil {
			return fmt.Errorf("invalid HTTP(S) URI: %w", err)
		}

		if parsedURL.Host == "" {
			return errors.New("invalid HTTP(S) URI: missing host")
		}

		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("unsupported protocol: %s", parsedURL.Scheme)
		}

	} else if strings.HasPrefix(repo, "git@") {
		if !strings.Contains(repo, ":") {
			return errors.New("invalid SSH URI format")
		}

		parts := strings.Split(repo, ":")
		if len(parts) != 2 {
			return errors.New("invalid SSH URI format")
		}

		if !strings.HasPrefix(parts[0], "git@") {
			return errors.New("SSH URI must start with 'git@'")
		}

	} else if strings.HasPrefix(repo, "ssh://") {
		parsedURL, err := url.Parse(repo)
		if err != nil {
			return fmt.Errorf("invalid SSH URI: %w", err)
		}

		if parsedURL.Host == "" {
			return errors.New("invalid SSH URI: missing host")
		}

	} else if strings.HasPrefix(repo, "file://") {
		return errors.New("file:// protocol is not allowed")

	} else {
		if strings.Contains(repo, "..") {
			return errors.New("repository path contains directory traversal")
		}
	}

	dangerousChars := []string{";", "|", "&", "`", "$", "(", ")", "<", ">", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(repo, char) {
			return fmt.Errorf("repository URI contains dangerous character: %q", char)
		}
	}

	return nil
}

func ValidateGitRepository(path string) error {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("not a git repository")
		}
		return fmt.Errorf("failed to check git repository: %w", err)
	}

	if !info.IsDir() {
		return errors.New(".git is not a directory")
	}

	return nil
}
