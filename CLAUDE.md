# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

fzfg is a CLI tool that wraps [fzf](https://github.com/junegunn/fzf) with a centralized YAML-based configuration system. It parses modular YAML configs defining fzf commands, options, and profiles, combines them into a single `config.yaml`, validates against fzf option definitions, and outputs formatted shell code. Fork of `krakozaure/fzg` (upstream remote).

## Commands (via Justfile)

```bash
just build       # Build binary to ./dist/fzfg
just run <ARGS>  # Run with args (e.g., just run -c fd_files -o preview)
just test        # Run Go unit tests + functional shell tests
just test-unit   # Run Go unit tests only (gotestsum)
just install     # Build + install to ~/.local/bin + copy configs to ~/.config/fzfg
just lint        # Run golangci-lint
just fmt         # Format Go source
```

Single external dependency: `gopkg.in/yaml.v2`. Logging: `charmbracelet/log`. Display: `charmbracelet/lipgloss`. Go 1.25.

## Architecture

Entry point: `cmd/fzfg/main.go`.

**`internal/` package**:
- `cli.go` — Flag definitions (`-q` quiet, `-r` raw, `-v` validate, `-init` pipeline, `-c` command, `-o` options, `-p` profile)
- `config.go` — Config loading, module discovery/merging, output formatting. `LoadConfig()` preprocesses includes, parses combined YAML, discovers module sections (unknown top-level keys containing commands/options/profiles), and merges them into top-level maps. Logger config is parsed from the `logger:` top-level key.
- `include.go` — `!include` preprocessing. Reads `fzfg.yaml`, expands `!include <glob>` directives by nesting module content under the parent key, writes combined `config.yaml`.
- `command.go` — Parses command entries (string or YAML sequence) into flat string slices.
- `options.go` — Parses option maps with special handling for `preview` (space-joined), `bind` (always quoted), booleans (`--flag`/`--no-flag`).
- `validate.go` — Three-phase validation: type/value checking, cross-option effect detection, conditional relationship checks. Loads definitions from `configs/definitions/options.yaml`.
- `logger.go` — Logging system using charmbracelet/log. Parses logger config from YAML, creates console/file/HTTP loggers with configurable levels, formats, and prefixes.
- `init.go` — `--init` pipeline: 6 sequential steps (start, config, validate, rsc-load, env-load, env-set) with per-step timing, config snapshots, and a summary table using lipgloss.

## Config System

**Config file resolution**: `FZFG_CONF` env var > `~/.config/fzfg/fzfg.yaml` > `./fzfg.yaml`.

**Modular structure**: `configs/fzfg.yaml` defines global defaults (logger, bindings, previews, option presets). Module files in `configs/modules/` define domain-specific commands/options/profiles using `!include` directives. On every run, all files are combined into `configs/config.yaml`.

**Definitions**: `configs/definitions/options.yaml` contains complete fzf option definitions (types, defaults, effects, conditions). `configs/definitions/keybindings.yaml` contains key names, events, actions, and default bindings.

**Init pipeline** (`-init` flag): Runs start > config > validate > rsc-load > env-load > env-set with timing data, then prints a summary table with final FZF_DEFAULT_COMMAND and FZF_DEFAULT_OPTS values.

**Output modes**:
- Default: `export FZF_DEFAULT_COMMAND="..."` / `export FZF_DEFAULT_OPTS="..."`
- Raw (`-r`): raw values separated by `\x1E`

## Testing

- Go unit tests: `internal/*_test.go` — table-driven tests for command/options parsing, include preprocessing, config loading, module merging, validation (type/enum/effects/conditions), logging, and init pipeline
- Functional tests: `scripts/tests` — end-to-end shell tests
