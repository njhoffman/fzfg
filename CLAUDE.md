# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

fzfg is a CLI tool that wraps [fzf](https://github.com/junegunn/fzf) with a centralized YAML-based configuration system. It parses modular YAML configs defining fzf commands, options, and profiles, combines them into a single `config.yaml`, validates against fzf option definitions, detects terminal/tmux layout for auto-sizing, and outputs formatted shell code. Fork of `krakozaure/fzg` (upstream remote).

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

## Architecture

Entry point: `cmd/fzfg/main.go`. Dependencies: `gopkg.in/yaml.v2`, `charmbracelet/log`, `charmbracelet/lipgloss`, `charmbracelet/x/term`. Go 1.25.

**`internal/` package**:
- `cli.go` — Flags: `-q` quiet, `-r` raw, `-v` validate, `--init[=STEP]` pipeline, `--debug=MODES`, `--module=NAME`, `-c` command, `-o` options, `-p` profile. `--init` is preprocessed before `flag.Parse()` to support both bare and `=value` forms.
- `config.go` — Config loading, module discovery/merging. Known top-level YAML keys: commands, options, profiles, bindings, previews, logger, layout.
- `include.go` — `!include` preprocessing for modular YAML configs.
- `command.go` — Parses command entries into flat string slices.
- `options.go` — Parses option maps with special handling for `preview`, `bind`, booleans.
- `validate.go` — Three-phase validation: type/enum checking, cross-option effects, conditional relationships.
- `logger.go` — charmbracelet/log setup from YAML config (console/file/HTTP outputs).
- `layout.go` — Terminal detection (`charmbracelet/x/term`), tmux state (pane count via `tmux list-panes`), auto-sized preview calculation ported from auto-sized-fzf (aspect ratio threshold, clamped percentage formula). Tmux popup vs split decision based on pane count and terminal width.
- `init.go` — Pipeline steps: start (terminal detect, config locate), config (load+merge), validate, rsc-load (fzfrc), env-load (FZF env vars), env-set (generate final values). Supports `--init=STEP` to run up to a specific step.
- `debug.go` — `--debug=MODE` output: summary (step statuses), timings (per-step with bar chart), diffs (config vs fzf defaults), envs (final FZF env vars + fzfrc content), trace (per-step config snapshots).

## Config System

**Precedence**: app defaults (fzf built-in) -> fzfrc (`FZF_DEFAULT_OPTS_FILE` / `~/.fzfrc`) -> config defaults -> module config.

**Config file resolution**: `FZFG_CONF` env > `~/.config/fzfg/fzfg.yaml` > `./fzfg.yaml`.

**Module selection**: `--module=NAME` flag or `FZF_MODULE` env var. Default: `files`.

**Definitions**: `configs/definitions/options.yaml` (fzf option types/defaults/effects), `configs/definitions/keybindings.yaml` (keys/events/actions/defaults).

## Usage from Shell (scripts/init.sh)

```bash
# Run individual pipeline steps
fzfg --init=start
fzfg --init=config
fzfg --init=validate

# Run full pipeline with debug output
fzfg --init -p view_files --debug=summary,timings,envs

# Normal usage
source <(fzfg -p view_files); fzf
```
