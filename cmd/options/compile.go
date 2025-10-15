package options

type Compile struct {
	Files  []string `long:"--file" short:"-f" help:"a list of files to be compiled like: -f a.proto -f b.proto"`
	Output string   `long:"--out" short:"-o" help:"output directory where the compiled files should be saved"`
	Method string   `long:"--method" help:"compilation method (fast-codec, pretty-codec) with fast-codec being default"`
	Help   bool     `long:"help" help:"shows help"`
}
