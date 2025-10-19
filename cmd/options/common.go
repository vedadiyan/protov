package options

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/vedadiyan/protov/internal/compiler"
)

const (
	MaxFilenameLength = 255
	MaxPathLength     = 4096
	CommandTimeout    = 30 * time.Second
	MaxFileSize       = 100 * 1024 * 1024 // 100MB
)

var (
	ErrInvalidPath      = errors.New("invalid path")
	ErrFileNotFound     = errors.New("file not found")
	ErrFileTooLarge     = errors.New("file exceeds maximum size")
	ErrCommandTimeout   = errors.New("command execution timeout")
	ErrToolNotFound     = errors.New("required tool not found")
	ErrEmptyData        = errors.New("empty data")
	ErrInvalidFilename  = errors.New("invalid filename")
	ErrNotWritable      = errors.New("directory not writable")
	ErrInvalidExtension = errors.New("invalid file extension")
)

// FileValidator provides file validation utilities
type FileValidator struct{}

// ValidateFilePath validates a file path for security and correctness
func (fv *FileValidator) ValidateFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty path", ErrInvalidPath)
	}

	cleaned := filepath.Clean(path)

	if len(cleaned) > MaxPathLength {
		return fmt.Errorf("%w: path too long (%d chars)", ErrInvalidPath, len(cleaned))
	}

	if strings.Contains(path, "\x00") {
		return fmt.Errorf("%w: path contains null byte", ErrInvalidPath)
	}

	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return fmt.Errorf("%w: cannot resolve absolute path: %v", ErrInvalidPath, err)
	}

	if strings.Contains(absPath, "..") {
		return fmt.Errorf("%w: path contains directory traversal", ErrInvalidPath)
	}

	return nil
}

// ValidateFileExists checks if a file exists and is accessible
func (fv *FileValidator) ValidateFileExists(path string) error {
	if err := fv.ValidateFilePath(path); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return fmt.Errorf("cannot access file %q: %w", path, err)
	}

	if info.IsDir() {
		return fmt.Errorf("%w: path is a directory, not a file", ErrInvalidPath)
	}

	if info.Size() > MaxFileSize {
		return fmt.Errorf("%w: %s (size: %d bytes)", ErrFileTooLarge, path, info.Size())
	}

	return nil
}

// ValidateProtoFile validates that a file is a .proto file
func (fv *FileValidator) ValidateProtoFile(path string) error {
	if err := fv.ValidateFileExists(path); err != nil {
		return err
	}

	if ext := filepath.Ext(path); ext != ".proto" {
		return fmt.Errorf("%w: expected .proto, got %q", ErrInvalidExtension, ext)
	}

	return nil
}

// ValidateOutputPath validates an output directory path
func (fv *FileValidator) ValidateOutputPath(path string) error {
	if path == "" {
		return fmt.Errorf("%w: empty output path", ErrInvalidPath)
	}

	cleaned := filepath.Clean(path)

	if len(cleaned) > MaxPathLength {
		return fmt.Errorf("%w: path too long (%d chars)", ErrInvalidPath, len(cleaned))
	}

	if strings.Contains(path, "\x00") {
		return fmt.Errorf("%w: path contains null byte", ErrInvalidPath)
	}

	// If directory exists, verify it's writable
	if info, err := os.Stat(cleaned); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%w: path exists but is not a directory", ErrInvalidPath)
		}

		testFile := filepath.Join(cleaned, ".write_test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return fmt.Errorf("%w: %v", ErrNotWritable, err)
		}
		os.Remove(testFile)
	}

	return nil
}

// FileIO handles safe file operations
type FileIO struct {
	validator *FileValidator
}

// NewFileIO creates a new FileIO instance
func NewFileIO() *FileIO {
	return &FileIO{
		validator: &FileValidator{},
	}
}

// ReadFile reads a file safely with validation
func (fio *FileIO) ReadFile(path string) ([]byte, error) {
	if err := fio.validator.ValidateFilePath(path); err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return nil, fmt.Errorf("cannot stat file: %w", err)
	}

	if info.Size() > MaxFileSize {
		return nil, fmt.Errorf("%w: %s", ErrFileTooLarge, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	return data, nil
}

// WriteFile writes data to a file safely with validation
func (fio *FileIO) WriteFile(path string, data []byte, perm os.FileMode) error {
	if len(data) == 0 {
		return ErrEmptyData
	}

	if err := fio.validator.ValidateFilePath(path); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	return nil
}

// EnsureDirectory creates a directory if it doesn't exist
func (fio *FileIO) EnsureDirectory(path string, perm os.FileMode) error {
	if err := fio.validator.ValidateFilePath(path); err != nil {
		return err
	}

	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// SanitizeFilename removes potentially dangerous characters from filenames
func (fio *FileIO) SanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "\x00", "")
	name = strings.TrimSpace(name)

	if len(name) > MaxFilenameLength {
		name = name[:MaxFilenameLength]
	}

	return name
}

// ReadTemplateFile reads a template file from absolute path or PROTOV_HOME
func (fio *FileIO) ReadTemplateFile(file string) ([]byte, error) {
	cleaned := filepath.Clean(file)

	if filepath.IsAbs(cleaned) {
		return fio.ReadFile(cleaned)
	}

	basePath := os.Getenv("PROTOV_HOME")
	if basePath == "" {
		return nil, errors.New("PROTOV_HOME environment variable not set")
	}

	templatePath := filepath.Join(basePath, "templates", cleaned)
	return fio.ReadFile(templatePath)
}

// CommandRunner handles external command execution
type CommandRunner struct {
	timeout time.Duration
}

// NewCommandRunner creates a new CommandRunner
func NewCommandRunner(timeout time.Duration) *CommandRunner {
	if timeout == 0 {
		timeout = CommandTimeout
	}
	return &CommandRunner{timeout: timeout}
}

// Run executes a command with timeout and error handling
func (cr *CommandRunner) Run(name string, dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cr.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: %s", ErrCommandTimeout, name)
		}
		return fmt.Errorf("command %q failed: %w\nstderr: %s", name, err, stderr.String())
	}

	return nil
}

// RunWithOutput executes a command and returns its output
func (cr *CommandRunner) RunWithOutput(name string, dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cr.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%w: %s", ErrCommandTimeout, name)
		}
		return nil, fmt.Errorf("command %q failed: %w\noutput: %s", name, err, string(output))
	}

	return output, nil
}

// CheckTool verifies that a tool is available in PATH
func (cr *CommandRunner) CheckTool(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}
	return nil
}

// CheckTools verifies multiple tools are available
func (cr *CommandRunner) CheckTools(tools []string) error {
	for _, tool := range tools {
		if err := cr.CheckTool(tool); err != nil {
			return err
		}
	}
	return nil
}

// GoFormatter handles Go code formatting
type GoFormatter struct {
	runner *CommandRunner
}

// NewGoFormatter creates a new GoFormatter
func NewGoFormatter() *GoFormatter {
	return &GoFormatter{
		runner: NewCommandRunner(CommandTimeout),
	}
}

// Format formats a Go file using gofmt and goimports
func (gf *GoFormatter) Format(dir, fileName string) error {
	if err := gf.runner.CheckTools([]string{"gofmt", "goimports"}); err != nil {
		return err
	}

	if err := gf.runner.Run("gofmt", dir, "-w", fileName); err != nil {
		return fmt.Errorf("gofmt failed: %w", err)
	}

	if err := gf.runner.Run("goimports", dir, "-w", fileName); err != nil {
		return fmt.Errorf("goimports failed: %w", err)
	}

	return nil
}

// ProtoCompiler handles proto file compilation
type ProtoCompiler struct {
	fileIO    *FileIO
	validator *FileValidator
	formatter *GoFormatter
}

// NewProtoCompiler creates a new ProtoCompiler
func NewProtoCompiler() *ProtoCompiler {
	return &ProtoCompiler{
		fileIO:    NewFileIO(),
		validator: &FileValidator{},
		formatter: NewGoFormatter(),
	}
}

// CompileFile compiles a single proto file
func (pc *ProtoCompiler) CompileFile(protoPath, outputDir string) (*compiler.AST, error) {
	if err := pc.validator.ValidateProtoFile(protoPath); err != nil {
		return nil, fmt.Errorf("invalid proto file: %w", err)
	}

	if err := pc.validator.ValidateOutputPath(outputDir); err != nil {
		return nil, fmt.Errorf("invalid output directory: %w", err)
	}

	ast, err := compiler.Parse(protoPath)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if ast == nil {
		return nil, errors.New("parser returned nil AST")
	}

	for _, file := range ast.Files {
		if err := pc.compileAndWriteFile(file, outputDir); err != nil {
			return nil, fmt.Errorf("compilation error for %q: %w", file.FileName, err)
		}
	}

	return ast, nil
}

// compileAndWriteFile compiles and writes a single file
func (pc *ProtoCompiler) compileAndWriteFile(file *compiler.File, outputDir string) error {
	compiled, err := compiler.Compile(file)
	if err != nil {
		return fmt.Errorf("compiler error: %w", err)
	}

	if len(compiled) == 0 {
		return ErrEmptyData
	}

	dir := filepath.Join(outputDir, file.FilePath)
	if err := pc.fileIO.EnsureDirectory(dir, 0755); err != nil {
		return err
	}

	fileName := pc.fileIO.SanitizeFilename(fmt.Sprintf("%s.pb.go", file.FileName))
	if fileName == "" {
		return ErrInvalidFilename
	}

	filePath := filepath.Join(dir, fileName)
	if err := pc.fileIO.WriteFile(filePath, compiled, 0644); err != nil {
		return err
	}

	if err := pc.formatter.Format(dir, fileName); err != nil {
		return err
	}

	return nil
}

// TemplateProcessor handles template-based code generation
type TemplateProcessor struct {
	fileIO    *FileIO
	formatter *GoFormatter
}

// NewTemplateProcessor creates a new TemplateProcessor
func NewTemplateProcessor() *TemplateProcessor {
	return &TemplateProcessor{
		fileIO:    NewFileIO(),
		formatter: NewGoFormatter(),
	}
}

// ProcessTemplate processes a template file and generates output
func (tp *TemplateProcessor) ProcessTemplate(templatePath string, data interface{}, outputDir, outputName string) error {
	templateData, err := tp.fileIO.ReadTemplateFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	if len(templateData) == 0 {
		return fmt.Errorf("%w: template is empty", ErrEmptyData)
	}

	tmpl, err := template.New("codegen").Parse(string(templateData))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if out.Len() == 0 {
		return fmt.Errorf("%w: template produced no output", ErrEmptyData)
	}

	fileName := tp.fileIO.SanitizeFilename(outputName)
	if fileName == "" {
		return ErrInvalidFilename
	}

	outputPath := filepath.Join(outputDir, fileName)
	if err := tp.fileIO.WriteFile(outputPath, out.Bytes(), 0644); err != nil {
		return err
	}

	// Format if it's a Go file
	if strings.HasSuffix(fileName, ".go") {
		if err := tp.formatter.Format(outputDir, fileName); err != nil {
			return fmt.Errorf("formatting error: %w", err)
		}
	}

	return nil
}

// ProcessServiceCodeGeneration processes code generation for services
func (tp *TemplateProcessor) ProcessServiceCodeGeneration(file *compiler.File, ast *compiler.AST, outputDir string) error {
	for _, srv := range file.Services {
		for _, cg := range srv.CodeGeneration {
			baseName := filepath.Base(cg)
			outputName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

			if err := tp.ProcessTemplate(cg, ast, outputDir, outputName); err != nil {
				return fmt.Errorf("failed to process template %q: %w", cg, err)
			}
		}
	}
	return nil
}
