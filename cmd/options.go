package main

import (
	flaggy "github.com/vedadiyan/flaggy/pkg"
	"github.com/vedadiyan/protov/cmd/options"
)

type (
	Options struct {
		Install options.Install `long:"install" help:"installs protov for the current user"`
		Pull    options.Pull    `long:"pull" help:"pulls protobuffer and template dependencies from a remote repository"`
		Compile options.Compile `long:"compile" help:"compiles one or more protobuffer file to Go"`
		Module  options.Module  `long:"module" help:"module utility to build or containerize protobuffer files"`
		Help    bool            `long:"help" help:"shows help"`
	}
)

func (x *Options) Run() error {
	flaggy.PrintHelp()
	return nil
}
