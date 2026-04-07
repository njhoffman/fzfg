package main

import (
	"flag"
	"fmt"
	"os"

	fzfg "github.com/njhoffman/fzfg/internal"
)

func main() {
	fzfg.InitFlags()

	// --init mode: run full pipeline with debug output
	if fzfg.InitFlag {
		result, err := fzfg.RunInit()
		if err != nil {
			if !fzfg.QuietFlag {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(1)
		}
		// If command/options flags were provided, print the output
		if result.FinalCmd != "" || result.FinalOpts != "" {
			out := fzfg.ParseConfig(
				fzfg.Command(result.FinalCmd),
				nil,
			)
			// Re-parse from config for proper formatting
			config := result.Config
			if fzfg.ProfileFlag != "" {
				out = fzfg.ParseConfig(
					config.Profiles[fzfg.ProfileFlag].Command,
					config.Profiles[fzfg.ProfileFlag].Options,
				)
			} else {
				var cmd fzfg.Command
				var opts fzfg.Options
				if fzfg.CommandFlag != "" {
					cmd = config.Commands[fzfg.CommandFlag]
				}
				if fzfg.OptionsFlag != "" {
					opts = config.Options[fzfg.OptionsFlag]
				}
				out = fzfg.ParseConfig(cmd, opts)
			}
			fmt.Print(out)
		}
		os.Exit(0)
	}

	config, err := fzfg.LoadConfig()
	if err != nil {
		if !fzfg.QuietFlag {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}

	err_text, out_text := "", ""
	if fzfg.CommandFlag != "" && fzfg.OptionsFlag != "" {
		err_text = fmt.Sprintf(
			"Invalid or missing configuration for '%s' or '%s'",
			fzfg.CommandFlag, fzfg.OptionsFlag,
		)
		out_text = fzfg.ParseConfig(
			config.Commands[fzfg.CommandFlag],
			config.Options[fzfg.OptionsFlag],
		)
	} else if fzfg.CommandFlag != "" {
		err_text = fmt.Sprintf(
			"Invalid or missing command configuration for '%s'",
			fzfg.CommandFlag,
		)
		out_text = fzfg.ParseConfig(
			config.Commands[fzfg.CommandFlag],
			nil,
		)
	} else if fzfg.OptionsFlag != "" {
		err_text = fmt.Sprintf(
			"Invalid or missing options configuration for '%s'",
			fzfg.OptionsFlag,
		)
		out_text = fzfg.ParseConfig(
			nil,
			config.Options[fzfg.OptionsFlag],
		)
	} else if fzfg.ProfileFlag != "" {
		err_text = fmt.Sprintf(
			"Invalid or missing profile configuration for '%s'",
			fzfg.ProfileFlag,
		)
		out_text = fzfg.ParseConfig(
			config.Profiles[fzfg.ProfileFlag].Command,
			config.Profiles[fzfg.ProfileFlag].Options,
		)
	} else {
		flag.Usage()
		os.Exit(1)
	}

	if out_text == "" {
		if !fzfg.QuietFlag {
			fmt.Fprintf(os.Stderr, "%s\n", err_text)
		}
		os.Exit(1)
	}
	fmt.Print(out_text)
}
