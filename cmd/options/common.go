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
	"github.com/vedadiyan/protov/internal/system/install"
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

func ValidateFilePath(path string) error {
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

func ValidateFileExists(path string) error {
	if err := ValidateFilePath(path); err != nil {
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

func ValidateProtoFile(path string) error {
	if err := ValidateFileExists(path); err != nil {
		return err
	}

	if ext := filepath.Ext(path); ext != ".proto" {
		return fmt.Errorf("%w: expected .proto, got %q", ErrInvalidExtension, ext)
	}

	return nil
}

func ValidateOutputPath(path string) error {
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

func ReadFile(path string) ([]byte, error) {
	if err := ValidateFilePath(path); err != nil {
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

func WriteFile(path string, data []byte, perm os.FileMode) error {
	if len(data) == 0 {
		return ErrEmptyData
	}

	if err := ValidateFilePath(path); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	return nil
}

func EnsureDirectory(path string, perm os.FileMode) error {
	if err := ValidateFilePath(path); err != nil {
		return err
	}

	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

func SanitizeFilename(name string) string {
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

func ReadTemplateFile(file string) ([]byte, error) {
	cleaned := filepath.Clean(file)

	if filepath.IsAbs(cleaned) {
		return ReadFile(cleaned)
	}

	basePath, err := install.ProtoPath()
	if err != nil {
		return nil, err
	}
	if basePath == "" {
		return nil, errors.New("protov environment variable not set")
	}

	templatePath := filepath.Join(basePath, "templates", cleaned)
	return ReadFile(templatePath)
}

func Exec(name string, dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
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

func CheckTool(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}
	return nil
}

func CheckTools(tools []string) error {
	for _, tool := range tools {
		if err := CheckTool(tool); err != nil {
			return err
		}
	}
	return nil
}

func Format(dir, fileName string) error {
	if err := CheckTools([]string{"gofmt", "goimports"}); err != nil {
		return err
	}

	if err := Exec("gofmt", dir, "-w", fileName); err != nil {
		return fmt.Errorf("gofmt failed: %w", err)
	}

	if err := Exec("goimports", dir, "-w", fileName); err != nil {
		return fmt.Errorf("goimports failed: %w", err)
	}

	return nil
}

func CompileFile(protoPath, outputDir string) (*compiler.AST, error) {
	if err := ValidateProtoFile(protoPath); err != nil {
		return nil, fmt.Errorf("invalid proto file: %w", err)
	}

	if err := ValidateOutputPath(outputDir); err != nil {
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
		if err := compileAndWriteFile(file, outputDir); err != nil {
			return nil, fmt.Errorf("compilation error for %q: %w", file.FileName, err)
		}

	}

	return ast, nil
}

func compileAndWriteFile(file *compiler.File, outputDir string) error {
	compiled, err := compiler.Compile(file)
	if err != nil {
		return fmt.Errorf("compiler error: %w", err)
	}

	if len(compiled) == 0 {
		return ErrEmptyData
	}

	dir := filepath.Join(outputDir, file.FilePath)
	if err := EnsureDirectory(dir, 0755); err != nil {
		return err
	}

	fileName := SanitizeFilename(fmt.Sprintf("%s.pb.go", file.FileName))
	if fileName == "" {
		return ErrInvalidFilename
	}

	filePath := filepath.Join(dir, fileName)
	if err := WriteFile(filePath, compiled, 0644); err != nil {
		return err
	}

	if err := Format(dir, fileName); err != nil {
		return err
	}

	return nil
}

func ProcessTemplate(templatePath string, data interface{}, outputDir, outputName string) error {
	templateData, err := ReadTemplateFile(templatePath)
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

	fileName := SanitizeFilename(outputName)
	if fileName == "" {
		return ErrInvalidFilename
	}

	outputPath := filepath.Join(outputDir, fileName)
	if err := WriteFile(outputPath, out.Bytes(), 0644); err != nil {
		return err
	}

	if strings.HasSuffix(fileName, ".go") {
		if err := Format(outputDir, fileName); err != nil {
			return fmt.Errorf("formatting error: %w", err)
		}
	}

	return nil
}

func ProcessServiceCodeGeneration(file *compiler.File, ast *compiler.AST, outputDir string) error {
	for _, srv := range file.Services {
		for _, cg := range srv.CodeGeneration {
			baseName := fmt.Sprintf("%s.%s", strings.ToLower(srv.Name), filepath.Base(cg))
			outputName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

			if err := ProcessTemplate(cg, struct {
				Source      string
				PackageName string
				Service     *compiler.Service
			}{file.Source, file.PackageName, srv}, outputDir, outputName); err != nil {
				return fmt.Errorf("failed to process template %q: %w", cg, err)
			}
		}
	}
	return nil
}
