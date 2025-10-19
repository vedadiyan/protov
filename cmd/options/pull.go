package options

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	flaggy "github.com/vedadiyan/flaggy/pkg"
	"github.com/vedadiyan/protov/internal/system/protoc"
)

var (
	ErrNoRepo          = errors.New("repository URI is required")
	ErrInvalidRepo     = errors.New("invalid repository URI")
	ErrProtoPathNotSet = errors.New("proto path not configured")
	ErrGitNotFound     = errors.New("git is not installed or not in PATH")
	ErrCloneFailed     = errors.New("git clone failed")
	ErrDirectoryCreate = errors.New("failed to create directory")
)

// Pull handles pulling external dependencies
type Pull struct {
	Proto    Proto    `long:"proto" help:"pulls protobuffer dependencies"`
	Template Template `long:"template" help:"pulls template dependencies"`
	Help     bool     `long:"help" help:"shows help"`
}

// Proto handles pulling proto dependencies
type Proto struct {
	Repo string `long:"--repo" help:"link to the repository"`
	Help bool   `long:"help" help:"shows help"`
}

// Run executes the proto dependency pull
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

// validate validates the proto pull configuration
func (p *Proto) validate() error {
	if len(p.Repo) == 0 {
		return ErrNoRepo
	}

	if err := validateRepoURI(p.Repo); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRepo, err)
	}

	return nil
}

// checkPrerequisites verifies required tools are available
func (p *Proto) checkPrerequisites() error {
	if err := CheckTool("git"); err != nil {
		return ErrGitNotFound
	}
	return nil
}

// pullRepository clones the repository to the proto include directory
func (p *Proto) pullRepository() error {
	protoPath, err := protoc.ProtoPath()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProtoPathNotSet, err)
	}

	if protoPath == "" {
		return ErrProtoPathNotSet
	}

	includeDir := filepath.Join(protoPath, "include")

	// Validate and create directory
	if err := EnsureDirectory(includeDir, 0755); err != nil {
		return fmt.Errorf("%w: %v", ErrDirectoryCreate, err)
	}

	// Verify directory is writable
	if err := ValidateOutputPath(includeDir); err != nil {
		return fmt.Errorf("include directory not writable: %w", err)
	}

	// Clone repository
	if err := p.cloneRepo(p.Repo, includeDir); err != nil {
		return fmt.Errorf("%w: %v", ErrCloneFailed, err)
	}

	return nil
}

// cloneRepo executes git clone command
func (p *Proto) cloneRepo(repoURI, targetDir string) error {
	args := []string{"clone", repoURI}

	// Add additional safety flags
	args = append(args, "--depth", "1") // Shallow clone for efficiency
	args = append(args, "--single-branch")

	if err := Run("git", targetDir, args...); err != nil {
		return err
	}

	return nil
}

// Template handles pulling template dependencies
type Template struct {
	Repo string `long:"--repo" help:"link to the template repository"`
	Help bool   `long:"help" help:"shows help"`
}

// Run executes the template dependency pull
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

// validate validates the template pull configuration
func (t *Template) validate() error {
	if len(t.Repo) == 0 {
		return ErrNoRepo
	}

	if err := validateRepoURI(t.Repo); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRepo, err)
	}

	return nil
}

// checkPrerequisites verifies required tools are available
func (t *Template) checkPrerequisites() error {
	if err := CheckTool("git"); err != nil {
		return ErrGitNotFound
	}
	return nil
}

// pullRepository clones the repository to the templates directory
func (t *Template) pullRepository() error {
	protoPath, err := protoc.ProtoPath()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProtoPathNotSet, err)
	}

	if protoPath == "" {
		return ErrProtoPathNotSet
	}

	templatesDir := filepath.Join(protoPath, "templates")

	// Validate and create directory
	if err := EnsureDirectory(templatesDir, 0755); err != nil {
		return fmt.Errorf("%w: %v", ErrDirectoryCreate, err)
	}

	// Verify directory is writable
	if err := ValidateOutputPath(templatesDir); err != nil {
		return fmt.Errorf("templates directory not writable: %w", err)
	}

	// Clone repository
	if err := t.cloneRepo(t.Repo, templatesDir); err != nil {
		return fmt.Errorf("%w: %v", ErrCloneFailed, err)
	}

	return nil
}

// cloneRepo executes git clone command
func (t *Template) cloneRepo(repoURI, targetDir string) error {
	args := []string{"clone", repoURI}

	// Add additional safety flags
	args = append(args, "--depth", "1") // Shallow clone for efficiency
	args = append(args, "--single-branch")

	if err := Run("git", targetDir, args...); err != nil {
		return err
	}

	return nil
}

// validateRepoURI validates a repository URI
func validateRepoURI(repo string) error {
	if repo == "" {
		return errors.New("empty repository URI")
	}

	// Trim whitespace
	repo = strings.TrimSpace(repo)

	// Check for null bytes
	if strings.Contains(repo, "\x00") {
		return errors.New("repository URI contains null byte")
	}

	// Validate URI format
	if strings.HasPrefix(repo, "http://") || strings.HasPrefix(repo, "https://") {
		// HTTP/HTTPS URI
		parsedURL, err := url.Parse(repo)
		if err != nil {
			return fmt.Errorf("invalid HTTP(S) URI: %w", err)
		}

		if parsedURL.Host == "" {
			return errors.New("invalid HTTP(S) URI: missing host")
		}

		// Security: prevent file:// protocol
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("unsupported protocol: %s", parsedURL.Scheme)
		}

	} else if strings.HasPrefix(repo, "git@") {
		// SSH URI (git@github.com:user/repo.git)
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
		// SSH URI (ssh://git@github.com/user/repo.git)
		parsedURL, err := url.Parse(repo)
		if err != nil {
			return fmt.Errorf("invalid SSH URI: %w", err)
		}

		if parsedURL.Host == "" {
			return errors.New("invalid SSH URI: missing host")
		}

	} else if strings.HasPrefix(repo, "file://") {
		// Block file:// protocol for security
		return errors.New("file:// protocol is not allowed")

	} else {
		// Assume local path or git URL without scheme
		// Additional validation for local paths
		if strings.Contains(repo, "..") {
			return errors.New("repository path contains directory traversal")
		}
	}

	// Check for common injection attempts
	dangerousChars := []string{";", "|", "&", "`", "$", "(", ")", "<", ">", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(repo, char) {
			return fmt.Errorf("repository URI contains dangerous character: %q", char)
		}
	}

	return nil
}

// ValidateGitRepository checks if a directory contains a valid git repository
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
