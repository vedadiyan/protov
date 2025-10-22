package options

import (
	"errors"
	"fmt"

	flaggy "github.com/vedadiyan/flaggy/pkg"

	"github.com/vedadiyan/protov/internal/compiler"
)

var (
	ErrNoFiles   = errors.New("no files provided")
	ErrNoOutput  = errors.New("output directory not specified")
	ErrBatchFail = errors.New("batch compilation failed")
)

type Compile struct {
	Files  []string `long:"--file" short:"-f" help:"a list of files to be compiled like: -f a.proto -f b.proto"`
	Output string   `long:"--out" short:"-o" help:"output directory where the compiled files should be saved"`
	Help   bool     `long:"help" help:"shows help"`
}

func (c *Compile) Run() error {
	if c.Help {
		flaggy.PrintHelp()
		return nil
	}

	if err := c.validate(); err != nil {
		flaggy.PrintHelp()
		return err
	}

	if err := c.checkPrerequisites(); err != nil {
		return fmt.Errorf("prerequisite check failed: %w", err)
	}

	return c.compileFiles()
}

func (c *Compile) validate() error {
	if len(c.Files) == 0 {
		return ErrNoFiles
	}

	for i, file := range c.Files {
		if err := ValidateProtoFile(file); err != nil {
			return fmt.Errorf("invalid file at index %d: %w", i, err)
		}
	}

	if len(c.Output) == 0 {
		return ErrNoOutput
	}

	if err := ValidateOutputPath(c.Output); err != nil {
		return fmt.Errorf("invalid output directory: %w", err)
	}

	return nil
}

func (c *Compile) checkPrerequisites() error {
	return CheckTools([]string{"gofmt", "goimports"})
}

func (c *Compile) compileFiles() error {
	var errors []error

	for _, file := range c.Files {
		if err := c.compileFile(file); err != nil {
			errors = append(errors, fmt.Errorf("failed to compile %q: %w", file, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%w: %v", ErrBatchFail, errors)
	}

	return nil
}

func (c *Compile) compileFile(protoPath string) error {
	ast, err := CompileFile(protoPath, c.Output)
	if err != nil {
		return err
	}

	return c.processCodeGeneration(ast)
}

func (c *Compile) processCodeGeneration(ast *compiler.AST) error {
	for _, file := range ast.Files {
		outputDir := c.Output
		if file.FilePath != "" {
			outputDir = c.Output + "/" + file.FilePath
		}

		if err := ProcessServiceCodeGeneration(file, ast, outputDir); err != nil {
			return fmt.Errorf("code generation failed for %q: %w", file.FileName, err)
		}
	}

	return nil
}
