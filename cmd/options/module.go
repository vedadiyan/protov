package options

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
	flaggy "github.com/vedadiyan/flaggy/pkg"
	"github.com/vedadiyan/protov/internal/compiler"
	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/modfile"
)

const (
	ConfigFilename     = "mod.yml"
	DockerfileTemplate = `
FROM alpine:latest
COPY . /srv
RUN chmod 755 /srv/app
WORKDIR /srv
CMD ./app
`
)

var (
	ErrConfigExists      = errors.New("module configuration already exists")
	ErrConfigNotFound    = errors.New("module configuration not found")
	ErrInvalidConfig     = errors.New("invalid configuration")
	ErrInvalidDependency = errors.New("invalid dependency format")
	ErrInvalidReplace    = errors.New("invalid replace format")
	ErrBuildFailed       = errors.New("build failed")
	ErrNoTag             = errors.New("tag is required")
	ErrProtoVHomeNotSet  = errors.New("protov environment variable not set")
)

type (
	ModuleConfig struct {
		Name         string            `yaml:"name"`
		Destination  string            `yaml:"destination"`
		Mod          string            `yaml:"mod"`
		GoVersion    string            `yaml:"go"`
		ProtoFiles   []string          `yaml:"protos"`
		Dependencies []string          `yaml:"dependencies"`
		Replacements []string          `yaml:"replacements"`
		MainTemplate []string          `yaml:"mainTemplate"`
		BuildFlags   []string          `yaml:"buildFlags"`
		Environment  map[string]string `yaml:"environment"`
		Tests        []string          `yaml:"tests"`
	}
	Config struct {
		Modules []ModuleConfig `yaml:"modules"`
	}
	Module struct {
		Init      ModuleInit      `long:"init" help:"initializes a new module"`
		Build     ModuleBuild     `long:"build" help:"builds the module into a standalone Go application"`
		Dockerize ModuleDockerize `long:"dockerize" help:"containerizes the module"`
		Help      bool            `long:"help" help:"shows help"`
	}
	ModuleInit struct {
	}
	ModuleBuild struct {
		Source bool `long:"--source" help:"builds the module into Go source code"`
		Help   bool `long:"help" help:"shows help"`
	}
	ModuleDockerize struct {
		Tag      string  `long:"--tag" help:"image tag name"`
		Builder  *string `long:"--builder" help:"specifies which tool to use to build the image"`
		Platform *string `long:"--platform" help:"specifies the platform for which the image must be built"`
		Buildx   bool    `long:"--buildx" help:"use buildx to build the image"`
		Help     bool    `long:"help" help:"shows help"`
	}
)

func (mod *Module) Run() error {
	flaggy.PrintHelp()
	return nil
}

func (c *Config) Validate() error {
	if len(c.Modules) == 0 {
		return fmt.Errorf("%w: no modules defined", ErrInvalidConfig)
	}

	for i, module := range c.Modules {
		if err := module.Validate(); err != nil {
			return fmt.Errorf("module at index %d: %w", i, err)
		}
	}

	return nil
}

func (mc *ModuleConfig) Validate() error {
	if mc.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidConfig)
	}
	if mc.Destination == "" {
		return fmt.Errorf("%w: destination is required", ErrInvalidConfig)
	}
	if mc.Mod == "" {
		return fmt.Errorf("%w: mod is required", ErrInvalidConfig)
	}
	if mc.GoVersion == "" {
		return fmt.Errorf("%w: go version is required", ErrInvalidConfig)
	}

	for _, dep := range mc.Dependencies {
		if _, err := ParseDependency(dep); err != nil {
			return fmt.Errorf("invalid dependency %q: %w", dep, err)
		}
	}

	for _, repl := range mc.Replacements {
		if err := validateReplacement(repl); err != nil {
			return fmt.Errorf("invalid replacement %q: %w", repl, err)
		}
	}

	return nil
}

func (mi *ModuleInit) Run() error {
	if _, err := os.Stat(ConfigFilename); err == nil {
		return ErrConfigExists
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check config file: %w", err)
	}

	config := mi.createDefaultConfig()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("generated invalid config: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := WriteFile(ConfigFilename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (mi *ModuleInit) createDefaultConfig() *Config {
	return &Config{
		Modules: []ModuleConfig{
			{
				Name:        "app",
				Destination: "./out/app",
				Mod:         "org/com/app",
				GoVersion:   strings.TrimPrefix(runtime.Version(), "go"),
				Dependencies: []string{
					"github.com/vedadiyan/protolizer v0.0.1",
				},
				Environment: map[string]string{
					"GOOS":   runtime.GOOS,
					"GOARCH": runtime.GOARCH,
				},
			},
		},
	}
}

func (mb *ModuleBuild) Run() error {
	if mb.Help {
		flaggy.PrintHelp()
		return nil
	}

	config, err := mb.loadConfig()
	if err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return Build(config, mb.Source)
}

func (mb *ModuleBuild) loadConfig() (*Config, error) {
	data, err := ReadFile(ConfigFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func (md *ModuleDockerize) Run() error {
	if md.Help {
		flaggy.PrintHelp()
		return nil
	}

	if len(md.Tag) == 0 {
		flaggy.PrintHelp()
		return ErrNoTag
	}

	if strings.ContainsAny(md.Tag, " \t\n\r") {
		return fmt.Errorf("invalid tag: contains whitespace")
	}

	config, err := md.loadConfig()
	if err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if err := Build(config, false); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	return md.dockerizeModules(config)
}

func (md *ModuleDockerize) loadConfig() (*Config, error) {
	data, err := ReadFile(ConfigFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func (md *ModuleDockerize) dockerizeModules(config *Config) error {
	builder := md.getBuilder()
	buildxFlag := md.getBuildxFlag()
	platform := md.getPlatform()

	for _, module := range config.Modules {
		if err := md.dockerizeModule(module, builder, buildxFlag, platform); err != nil {
			return fmt.Errorf("failed to dockerize module %q: %w", module.Name, err)
		}
	}

	return nil
}

func (md *ModuleDockerize) dockerizeModule(module ModuleConfig, builder, buildxFlag, platform string) error {
	dockerfilePath := filepath.Join(module.Destination, "Dockerfile")
	if err := WriteFile(dockerfilePath, []byte(DockerfileTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	args := md.buildDockerArgs(buildxFlag, platform)
	if err := Exec(builder, module.Destination, args...); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	if err := os.RemoveAll(module.Destination); err != nil {
		return fmt.Errorf("failed to cleanup: %w", err)
	}

	return nil
}

func (md *ModuleDockerize) getBuilder() string {
	if md.Builder != nil && *md.Builder != "" {
		return *md.Builder
	}
	return "docker"
}

func (md *ModuleDockerize) getBuildxFlag() string {
	if md.Buildx {
		return "buildx"
	}
	return ""
}

func (md *ModuleDockerize) getPlatform() string {
	if md.Platform != nil && *md.Platform != "" {
		return "--platform=" + *md.Platform
	}
	return ""
}

func (md *ModuleDockerize) buildDockerArgs(buildxFlag, platform string) []string {
	args := []string{"build"}

	if buildxFlag != "" {
		args = append(args, buildxFlag)
	}

	if platform != "" {
		args = append(args, platform)
	}

	args = append(args, "-t", md.Tag, ".")

	if md.Buildx {
		args = append(args, "--output", "type=docker")
	}

	return args
}

func Build(config *Config, sourceOnly bool) error {
	if config == nil {
		return fmt.Errorf("%w: config is nil", ErrInvalidConfig)
	}

	for _, module := range config.Modules {
		if err := buildModule(module, sourceOnly); err != nil {
			return fmt.Errorf("failed to build module %q: %w", module.Name, err)
		}
	}

	return nil
}

func buildModule(module ModuleConfig, sourceOnly bool) error {
	if err := EnsureDirectory(module.Destination, 0755); err != nil {
		return err
	}

	if err := createGoMod(module); err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	files, err := compileProtoFiles(module)
	if err != nil {
		return fmt.Errorf("proto compilation failed: %w", err)
	}

	if err := generateMainFiles(module, files); err != nil {
		return fmt.Errorf("main generation failed: %w", err)
	}

	if !sourceOnly {
		if err := buildBinary(module); err != nil {
			return fmt.Errorf("binary build failed: %w", err)
		}
	}

	return nil
}

func createGoMod(module ModuleConfig) error {
	mod := new(modfile.File)

	if err := mod.AddModuleStmt(module.Mod); err != nil {
		return fmt.Errorf("failed to add module statement: %w", err)
	}

	if err := mod.AddGoStmt(module.GoVersion); err != nil {
		return fmt.Errorf("failed to add go version: %w", err)
	}

	for _, dep := range module.Dependencies {
		if err := addDependency(mod, dep); err != nil {
			return err
		}
	}

	for _, repl := range module.Replacements {
		if err := addReplacement(mod, repl); err != nil {
			return err
		}
	}

	modBytes, err := mod.Format()
	if err != nil {
		return fmt.Errorf("failed to format go.mod: %w", err)
	}

	modPath := filepath.Join(module.Destination, "go.mod")
	return WriteFile(modPath, modBytes, 0644)
}

func addDependency(mod *modfile.File, dep string) error {
	parts, err := ParseDependency(dep)
	if err != nil {
		return err
	}

	if err := mod.AddRequire(parts[0], parts[1]); err != nil {
		return fmt.Errorf("failed to add dependency %q: %w", dep, err)
	}

	return nil
}

func addReplacement(mod *modfile.File, repl string) error {
	segments := strings.Split(repl, "=>")
	if len(segments) != 2 {
		return fmt.Errorf("%w: expected format 'old => new', got %q", ErrInvalidReplace, repl)
	}

	oldDep, err := ParseDependency(strings.TrimSpace(segments[0]))
	if err != nil {
		return fmt.Errorf("%w in old dependency: %v", ErrInvalidReplace, err)
	}

	newDep, err := ParseDependency(strings.TrimSpace(segments[1]))
	if err != nil {
		return fmt.Errorf("%w in new dependency: %v", ErrInvalidReplace, err)
	}

	if err := mod.AddReplace(oldDep[0], oldDep[1], newDep[0], newDep[1]); err != nil {
		return fmt.Errorf("failed to add replacement: %w", err)
	}

	return nil
}

func compileProtoFiles(module ModuleConfig) ([]*compiler.File, error) {
	if len(module.ProtoFiles) == 0 {
		return []*compiler.File{}, nil
	}

	var allFiles []*compiler.File

	for _, protoPath := range module.ProtoFiles {
		if err := ValidateProtoFile(protoPath); err != nil {
			return nil, fmt.Errorf("invalid proto file %q: %w", protoPath, err)
		}

		ast, err := CompileFile(protoPath, module.Destination)
		if err != nil {
			return nil, fmt.Errorf("failed to compile %q: %w", protoPath, err)
		}

		if err := processCodeGeneration(ast, module.Destination); err != nil {
			return nil, err
		}

		allFiles = append(allFiles, ast.Files...)
	}

	return allFiles, nil
}

func processCodeGeneration(ast *compiler.AST, destination string) error {
	for _, file := range ast.Files {
		outputDir := filepath.Join(destination, file.FilePath)
		if err := ProcessServiceCodeGeneration(file, ast, outputDir); err != nil {
			return fmt.Errorf("code generation failed: %w", err)
		}
	}
	return nil
}

func generateMainFiles(module ModuleConfig, files []*compiler.File) error {
	if len(module.MainTemplate) == 0 {
		return nil
	}

	cmdDir := filepath.Join(module.Destination, "cmd")
	if err := EnsureDirectory(cmdDir, 0755); err != nil {
		return err
	}

	for _, templatePath := range module.MainTemplate {
		baseName := filepath.Base(templatePath)
		outputName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

		if err := ProcessTemplate(templatePath, files, cmdDir, outputName); err != nil {
			return fmt.Errorf("failed to process main template %q: %w", templatePath, err)
		}
	}

	return nil
}

func buildBinary(module ModuleConfig) error {
	tmpDir := os.TempDir()
	tmpBinary := filepath.Join(tmpDir, uuid.New().String())

	buildArgs := []string{"build", "-o", tmpBinary, "./cmd/"}
	buildArgs = append(buildArgs, module.BuildFlags...)

	if err := setEnvironment(module.Environment); err != nil {
		return err
	}

	if err := Exec("go", module.Destination, "mod", "tidy"); err != nil {
		return fmt.Errorf("%w: %v", ErrBuildFailed, err)
	}

	if err := Exec("go", module.Destination, buildArgs...); err != nil {
		return fmt.Errorf("%w: %v", ErrBuildFailed, err)
	}

	if err := cleanupAndMoveBinary(module, tmpBinary); err != nil {
		return err
	}

	return nil
}

func setEnvironment(env map[string]string) error {
	for key, value := range env {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}
	return nil
}

func cleanupAndMoveBinary(module ModuleConfig, tmpBinary string) error {
	entries, err := os.ReadDir(module.Destination)
	if err != nil {
		return fmt.Errorf("failed to read destination directory: %w", err)
	}

	for _, entry := range entries {
		path := filepath.Join(module.Destination, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %q: %w", path, err)
		}
	}

	binaryName := getBinaryName(module.Environment["GOOS"])
	finalPath := filepath.Join(module.Destination, binaryName)

	if err := os.Rename(tmpBinary, finalPath); err != nil {
		return fmt.Errorf("failed to move binary: %w", err)
	}

	if err := os.Chmod(finalPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}

func getBinaryName(goos string) string {
	switch goos {
	case "windows":
		return "app.exe"
	case "darwin":
		return "app.dmg"
	default:
		return "app"
	}
}

func ParseDependency(dep string) ([2]string, error) {
	dep = strings.TrimSpace(dep)
	if dep == "" {
		return [2]string{}, fmt.Errorf("%w: empty dependency", ErrInvalidDependency)
	}

	segments := strings.Fields(dep)
	if len(segments) > 2 {
		return [2]string{}, fmt.Errorf("%w: expected 'path' or 'path version', got %q", ErrInvalidDependency, dep)
	}

	path := segments[0]
	version := "v0.0.0"

	if len(segments) == 2 {
		version = segments[1]
		if !strings.HasPrefix(version, "v") && version != "latest" {
			return [2]string{}, fmt.Errorf("%w: version must start with 'v' or be 'latest', got %q", ErrInvalidDependency, version)
		}
	}

	return [2]string{path, version}, nil
}

func validateReplacement(repl string) error {
	segments := strings.Split(repl, "=>")
	if len(segments) != 2 {
		return fmt.Errorf("%w: expected format 'old => new'", ErrInvalidReplace)
	}

	if _, err := ParseDependency(strings.TrimSpace(segments[0])); err != nil {
		return fmt.Errorf("%w: invalid old dependency", ErrInvalidReplace)
	}

	if _, err := ParseDependency(strings.TrimSpace(segments[1])); err != nil {
		return fmt.Errorf("%w: invalid new dependency", ErrInvalidReplace)
	}

	return nil
}

func ParseGoFile(code []byte) (string, error) {
	if len(code) == 0 {
		return "", errors.New("empty Go source code")
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "temp.go", code, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("failed to parse Go file: %w", err)
	}

	if file.Name == nil {
		return "", errors.New("package name not found")
	}

	return file.Name.Name, nil
}
