package options

import (
	"fmt"
	"os"
	"os/exec"

	flaggy "github.com/vedadiyan/flaggy/pkg"
	"github.com/vedadiyan/protov/internal/system/protoc"
)

type Proto struct {
	Repo string `long:"--repo" help:"link to the repository"`
	Help bool   `long:"help" help:"shows help"`
}

type Template struct {
	Repo string `long:"--repo" help:"link to the template repository"`
	Help bool   `long:"help" help:"shows help"`
}

type Pull struct {
	Proto Proto `long:"proto" help:"pulls protobuffer dependencies"`
	Help  bool  `long:"help" help:"shows help"`
}

func (x *Proto) Run() error {
	if x.Help {
		flaggy.PrintHelp()
		return nil
	}
	if len(x.Repo) == 0 {
		flaggy.PrintHelp()
		return fmt.Errorf("repo uri is required")
	}

	protoPath, err := protoc.ProtoPath()
	if err != nil {
		return err
	}

	homeDir := fmt.Sprintf("%s/include/", protoPath)
	cmd := exec.Command("git", "clone", x.Repo)
	cmd.Dir = homeDir
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (x *Template) Run() error {
	if x.Help {
		flaggy.PrintHelp()
		return nil
	}

	if len(x.Repo) == 0 {
		flaggy.PrintHelp()
		return fmt.Errorf("repo uri is required")
	}

	protoPath, err := protoc.ProtoPath()
	if err != nil {
		return err
	}

	homeDir := fmt.Sprintf("%s/templates/", protoPath)
	cmd := exec.Command("git", "clone", x.Repo)
	if err := os.MkdirAll(homeDir, os.ModePerm); err != nil {
		return err
	}
	cmd.Dir = homeDir
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
