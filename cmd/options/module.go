package options

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/google/uuid"
	flaggy "github.com/vedadiyan/flaggy/pkg"
	"github.com/vedadiyan/protov/internal/compiler"
	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/modfile"
)

const FILENAME = "mod.yml"

const DOCKERFILE = `
FROM alpine:latest
COPY . /srv
RUN chmod 777 /srv/app
WORKDIR /srv
CMD ./app
`

type Config struct {
	Modules []ModuleConfig `yaml:"modules"`
}

type ModuleConfig struct {
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

type ModuleInit struct{}

type ModuleBuild struct {
	Source bool `long:"--source" help:"builds the module into Go source code"`
	Help   bool `long:"help" help:"shows help"`
}

type ModuleDockerize struct {
	Tag      string  `long:"--tag" help:"image tag name"`
	Builder  *string `long:"--builder" help:"specifies which tool to use to build the image"`
	Platform *string `long:"--platform" help:"specifies the platform for which the image must be built"`
	Buildx   bool    `long:"--buildx" help:"used buildx to build the image"`
	Help     bool    `long:"help" help:"shows help"`
}

type Module struct {
	Init      ModuleInit      `long:"init" help:"initializes a new module"`
	Build     ModuleBuild     `long:"build" help:"builds the module into a standalone Go application"`
	Dockerize ModuleDockerize `long:"dockerize" help:"containerizes the module"`
	Help      bool            `long:"help" help:"shows help"`
}

func (x *ModuleInit) Run() error {
	_, err := os.Stat(FILENAME)
	if err == nil {
		return fmt.Errorf("module file already exists")
	}
	if !os.IsNotExist(err) {
		return err
	}

	mod := ModuleConfig{
		Name:        "app",
		Destination: "/out/app",
		Mod:         "org/com/app",
		GoVersion:   strings.ReplaceAll(runtime.Version(), "go", ""),
		Dependencies: []string{
			"github.com/vedadiyan/protolizer",
		},
		Environment: map[string]string{
			"GOOS":   runtime.GOOS,
			"GOARCH": runtime.GOARCH,
		},
	}

	conf := new(Config)
	conf.Modules = append(conf.Modules, mod)

	out, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	return os.WriteFile(FILENAME, out, os.ModePerm)
}

func (x *ModuleBuild) Run() error {
	data, err := os.ReadFile(FILENAME)
	if err != nil || os.IsNotExist(err) {
		return err
	}

	conf := new(Config)

	if err := yaml.Unmarshal(data, &conf); err != nil {
		return err
	}

	return Build(conf, x.Source)
}

func (x *ModuleDockerize) Run() error {

	if len(x.Tag) == 0 {
		flaggy.PrintHelp()
		return fmt.Errorf("tag is required")
	}

	data, err := os.ReadFile(FILENAME)
	if err != nil || os.IsNotExist(err) {
		return err
	}

	conf := new(Config)

	if err := yaml.Unmarshal(data, &conf); err != nil {
		return err
	}

	if err := Build(conf, false); err != nil {
		return err
	}

	builder := "docker"

	if x.Builder != nil {
		builder = *x.Builder
	}
	buildx := ""
	tail := ""
	if x.Buildx {
		buildx = "buildx"
		tail = "--output type=docker"
	}
	platform := ""
	if x.Platform != nil {
		platform = "--platform=" + *x.Platform
	}

	for _, i := range conf.Modules {
		if err := os.WriteFile(filepath.Join(i.Destination, "DOCKERFILE"), []byte(DOCKERFILE), os.ModePerm); err != nil {
			return err
		}

		cmd := exec.Command(builder, "build", buildx, platform, platform, "-t", x.Tag, ".", tail)
		cmd.Dir = i.Destination
		if err := cmd.Run(); err != nil {
			return err
		}
		if err := os.RemoveAll(i.Destination); err != nil {
			return err
		}
	}
	return nil
}

func Build(conf *Config, source bool) error {
	for _, i := range conf.Modules {
		mod := new(modfile.File)
		if err := mod.AddModuleStmt(i.Mod); err != nil {
			return err
		}
		if err := mod.AddGoStmt(i.GoVersion); err != nil {
			return err
		}
		for _, r := range i.Dependencies {
			dep, err := GetDependency(r)
			if err != nil {
				return err
			}
			if err := mod.AddRequire(dep[0], dep[1]); err != nil {
				return err
			}
		}
		for _, r := range i.Replacements {
			segments := strings.Split(r, "=>")
			if len(segments) != 2 {
				return fmt.Errorf("bad replace format")
			}
			oldDep, err := GetDependency(segments[0])
			if err != nil {
				return err
			}
			newDep, err := GetDependency(segments[1])
			if err != nil {
				return err
			}
			if err := mod.AddReplace(oldDep[0], oldDep[1], newDep[0], newDep[1]); err != nil {
				return err
			}
		}
		modBytes, err := mod.Format()
		if err != nil {
			return err
		}

		if err := os.MkdirAll(i.Destination, os.ModePerm); err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(i.Destination, "go.mod"), modBytes, os.ModePerm); err != nil {
			return err
		}

		files := make([]*compiler.File, 0)
		for _, x := range i.ProtoFiles {
			ast, err := compiler.Parse(x)
			if err != nil {
				return err
			}
			for _, f := range ast.Files {
				files = append(files, f)
				compiled, err := compiler.Compile(f)
				if err != nil {
					return err
				}
				dir := filepath.Join(i.Destination, f.FilePath)
				if err := os.MkdirAll(dir, os.ModePerm); err != nil {
					return err
				}
				fileName := fmt.Sprintf("%s.pb.go", f.FileName)
				path := filepath.Join(dir, fileName)
				if err := os.WriteFile(path, compiled, os.ModePerm); err != nil {
					return err
				}
				cmd := exec.Command("gofmt", "-w", fileName)
				cmd.Dir = dir
				if err := cmd.Run(); err != nil {
					return err
				}
				cmd = exec.Command("goimports", "-w", fileName)
				cmd.Dir = dir
				if err := cmd.Run(); err != nil {
					return err
				}

				for _, srv := range f.Services {
					for _, cg := range srv.CodeGeneration {
						data, err := ReadFile(cg)
						if err != nil {
							return err
						}
						template, err := template.New("temp").Parse(string(data))
						if err != nil {
							return err
						}
						out := bytes.NewBuffer([]byte{})
						if err := template.Execute(out, ast); err != nil {
							return err
						}
						_, fileName := filepath.Split(strings.ReplaceAll(cg, filepath.Ext(cg), ""))
						path := filepath.Join(dir, fileName)
						if err := os.WriteFile(path, out.Bytes(), os.ModePerm); err != nil {
							return err
						}
						cmd := exec.Command("gofmt", "-w", fileName)
						cmd.Dir = dir
						if err := cmd.Run(); err != nil {
							return err
						}
						cmd = exec.Command("goimports", "-w", fileName)
						cmd.Dir = dir
						if err := cmd.Run(); err != nil {
							return err
						}
					}
				}
			}
		}
		for _, x := range i.MainTemplate {
			data, err := ReadFile(x)
			if err != nil {
				return err
			}
			template, err := template.New("temp").Parse(string(data))
			if err != nil {
				return err
			}
			out := bytes.NewBuffer([]byte{})
			if err := template.Execute(out, files); err != nil {
				return err
			}
			_, fileName := filepath.Split(strings.ReplaceAll(x, filepath.Ext(x), ""))
			dir := filepath.Join(i.Destination, "cmd")
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return err
			}
			path := filepath.Join(dir, fileName)
			if err := os.WriteFile(path, out.Bytes(), os.ModePerm); err != nil {
				return err
			}
		}
		if !source {
			tmp := os.TempDir()
			id := uuid.New().String()
			cmd := exec.Command("go", "build", "-o", filepath.Join(tmp, id), "./cmd/")
			cmd.Dir = i.Destination
			if err := cmd.Run(); err != nil {
				return err
			}
			dir, err := os.ReadDir(i.Destination)
			if err != nil {
				return err
			}
			for _, x := range dir {
				if err := os.RemoveAll(filepath.Join(i.Destination, x.Name())); err != nil {
					return err
				}
			}
			ext := ""
			switch i.Environment["GOOS"] {
			case "windows":
				{
					ext = ".exe"
				}
			case "darwin":
				{
					ext = ".dmg"
				}
			}
			if err := os.Rename(filepath.Join(tmp, id), filepath.Join(i.Destination, fmt.Sprintf("app%s", ext))); err != nil {
				return err
			}
		}
	}
	return nil
}

func GetDependency(dep string) ([2]string, error) {
	segments := strings.Split(dep, " ")
	if len(segments) > 2 {
		return [2]string{}, fmt.Errorf("bad dependency format")
	}
	path := segments[0]
	version := "v0.0.0"
	if len(segments) == 2 {
		version = segments[1]
	}
	return [2]string{path, version}, nil
}

func ReadFile(file string) ([]byte, error) {
	if filepath.IsAbs(filepath.Clean(file)) {
		return os.ReadFile(file)
	}

	basePath := os.Getenv("PROTOV_HOME")
	return os.ReadFile(filepath.Join(basePath, "templates", file))
}

func ParsePath(code []byte) (string, error) {
	fileSet := token.NewFileSet()
	expr, err := parser.ParseFile(fileSet, "test.go", code, parser.ParseComments)
	if err != nil {
		return "", err
	}
	return expr.Name.Name, nil
}
