package options

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"text/template"

	"github.com/vedadiyan/protov/internal/compiler"
	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/modfile"
)

const FILENAME = "mod.yml"

type Config struct {
	Modules []ModuleConfig `yaml:"modules"`
}

type ModuleConfig struct {
	Name          string            `yaml:"name"`
	Destination   string            `yaml:"destination"`
	Mod           string            `yaml:"mod"`
	GoVersion     string            `yaml:"go"`
	ProtoFiles    []string          `yaml:"protos"`
	Dependencies  []string          `yaml:"dependencies"`
	Replacements  []string          `yaml:"replacements"`
	TemplateFiles []string          `yaml:"templateFiles"`
	BuildFlags    []string          `yaml:"buildFlags"`
	Environment   map[string]string `yaml:"environment"`
	Tests         []string          `yaml:"tests"`
}

type ModuleInit struct{}

type ModuleBuild struct {
	Source bool `long:"--source" help:"builds the module into Go source code"`
	Help   bool `long:"help" help:"shows help"`
}

type ModuleDockerize struct {
	Tag  string `long:"--tag" help:"image tag name"`
	Help bool   `long:"help" help:"shows help"`
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
		GoVersion:   runtime.Version(),
		ProtoFiles: []string{
			"",
		},
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
		for _, r := range i.Dependencies {
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

		if err := os.WriteFile(path.Join(i.Destination, "go.mod"), modBytes, os.ModePerm); err != nil {
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
				path := path.Join(i.Destination, f.FilePath, fmt.Sprintf("%s.pb.go", f.FileName))
				if err := os.WriteFile(path, compiled, os.ModePerm); err != nil {
					return err
				}
			}
		}
		for _, x := range i.TemplateFiles {
			_ = x
			template, err := template.New("").Parse("")
			if err != nil {
				return err
			}
			out := bytes.NewBuffer([]byte{})
			if err := template.Execute(out, files); err != nil {
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
