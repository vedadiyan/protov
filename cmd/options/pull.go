package options

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
