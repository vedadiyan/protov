package options

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
