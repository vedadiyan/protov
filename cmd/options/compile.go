package options

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/vedadiyan/protov/internal/compiler"
)

type Compile struct {
	Files  []string `long:"--file" short:"-f" help:"a list of files to be compiled like: -f a.proto -f b.proto"`
	Output string   `long:"--out" short:"-o" help:"output directory where the compiled files should be saved"`
	// Method string   `long:"--method" help:"compilation method (fast-codec, pretty-codec) with fast-codec being default"`
	Help bool `long:"help" help:"shows help"`
}

func (x *Compile) Run() error {
	for _, f := range x.Files {
		ast, err := compiler.Parse(f)
		if err != nil {
			return err
		}
		for _, file := range ast.Files {
			compiled, err := compiler.Compile(file)
			if err != nil {
				return err
			}
			dir := filepath.Join(x.Output, file.FilePath)
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return err
			}
			path := filepath.Join(dir, fmt.Sprintf("%s.pb.go", file.FileName))
			if err := os.WriteFile(path, compiled, os.ModePerm); err != nil {
				return err
			}
			for _, srv := range file.Services {
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
				}
			}
		}
	}
	return nil
}
