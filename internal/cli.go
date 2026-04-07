package fzfg

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	QuietFlag    bool
	RawFlag      bool
	ValidateFlag bool

	InitFlag   string // --init=step or --init (all steps)
	DebugFlag  string // --debug=summary,diffs,timings,envs
	ModuleFlag string // --module=files

	CommandFlag string
	OptionsFlag string
	ProfileFlag string
)

func InitFlags() {
	flag.BoolVar(
		&QuietFlag,
		"q",
		false,
		"Fail without printing errors but with exit code > 0 (default: false)",
	)
	flag.BoolVar(
		&RawFlag,
		"r",
		false,
		"Print raw value without variable name or quoting (default: false)",
	)
	flag.BoolVar(
		&ValidateFlag,
		"v",
		false,
		"Validate configuration options against fzf defaults (default: false)",
	)

	// --init is handled manually before flag.Parse() to support both
	// --init (bare, runs all steps) and --init=step (specific step).
	// See preprocessInitArg().
	flag.StringVar(
		&DebugFlag,
		"debug",
		"",
		"Debug output modes: summary,diffs,timings,envs,trace (comma-separated)",
	)
	flag.StringVar(
		&ModuleFlag,
		"module",
		"",
		"Module to use for commands/options/profiles (default: files, env: FZF_MODULE)",
	)

	flag.StringVar(
		&CommandFlag,
		"c",
		"",
		"Configuration key to use for the command",
	)
	flag.StringVar(
		&OptionsFlag,
		"o",
		"",
		"Configuration key to use for the options",
	)
	flag.StringVar(
		&ProfileFlag,
		"p",
		"",
		"Configuration key to use for the profile (command+options)",
	)

	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`USAGE: %s [-q] [-r] [-v] [-init[=STEP]] [-debug=MODES] [-module=NAME]
       %s [-c CMD -o OPTS | -c CMD | -o OPTS | -p PROFILE]

OPTIONS:
`,
			os.Args[0], os.Args[0],
		)
		flag.PrintDefaults()
	}

	// Preprocess --init before flag.Parse to handle bare --init
	preprocessInitArg()

	flag.Parse()

	// Resolve module from env if not set via flag
	if ModuleFlag == "" {
		ModuleFlag = os.Getenv("FZF_MODULE")
	}
	if ModuleFlag == "" {
		ModuleFlag = "files" // default module
	}
}

// preprocessInitArg handles the --init flag manually because it needs to
// support both --init (bare, all steps) and --init=step (specific step).
// It removes --init or --init=value from os.Args and sets InitFlag.
func preprocessInitArg() {
	var filtered []string
	for _, arg := range os.Args[1:] {
		if arg == "--init" || arg == "-init" {
			InitFlag = "all"
		} else if strings.HasPrefix(arg, "--init=") || strings.HasPrefix(arg, "-init=") {
			InitFlag = strings.SplitN(arg, "=", 2)[1]
		} else {
			filtered = append(filtered, arg)
		}
	}
	os.Args = append(os.Args[:1], filtered...)
}
