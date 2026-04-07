# fzf-repl

Interactive REPL for controlling [fzf](https://github.com/junegunn/fzf) via its `--listen` socket API.

## Quick Start

```bash
# Start fzf with a socket listener
export FZF_SOCK=/tmp/fzf.sock
fzf --listen=$FZF_SOCK

# In another terminal, start the REPL
fzf-repl
```

## Connection

fzf-repl connects to fzf via Unix socket or TCP. It resolves the connection in this order:

1. `-socket` flag
2. `-tcp` flag
3. `FZF_SOCK` environment variable
4. `FZF_PORT` environment variable

```bash
# Explicit socket
fzf-repl -socket /tmp/fzf.sock

# TCP connection
fzf-repl -tcp localhost:6266

# With API key
FZF_API_KEY=mykey fzf-repl
```

## REPL Commands

| Command | Description |
|---------|-------------|
| `help`, `?` | Show command reference |
| `quit`, `exit`, `q` | Exit the REPL |
| `ping` | Test connection to fzf |
| `info` | Show connection details |
| `state [limit] [offset]` | Pretty-print fzf state |
| `raw [limit]` | Show raw JSON state (colorized) |
| `actions` | List available fzf actions |
| `events` | List available fzf events |
| `keys` | List common key names |
| `<anything else>` | Send as action to fzf |

## Sending Actions

Any input that isn't a built-in command is sent directly to fzf as a POST action:

```
fzf> up
ok
fzf> reload(seq 100)
ok
fzf> change-query(hello)+change-prompt(search> )
ok
fzf> select-all+accept
ok
```

## Non-Interactive Mode

```bash
# Send a single action
fzf-repl -send "reload(seq 100)+change-prompt(hundred> )"

# Get state as JSON
fzf-repl -state | jq .

# Get state with custom limit
fzf-repl -state -limit 1000 | jq '.matches[].text'
```

## Tmux Integration

If tmux is available and no fzf socket is found, fzf-repl can create a pane:

```bash
# Create a tmux split pane running fzf with --listen
fzf-repl -create-pane

# With custom socket path
fzf-repl -create-pane -pane-socket /tmp/my-fzf.sock

# With extra fzf args
fzf-repl -create-pane -pane-args "--multi,--preview=cat {}"
```

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-socket` | `$FZF_SOCK` | Unix socket path |
| `-tcp` | `$FZF_PORT` | TCP address |
| `-api-key` | `$FZF_API_KEY` | Authentication key |
| `-send` | | Send action and exit |
| `-state` | | Get JSON state and exit |
| `-limit` | 100 | Item limit for state queries |
| `-log-level` | info | Log verbosity (debug/info/warn/error) |
| `-create-pane` | | Create tmux pane with fzf listener |

## fzf Socket API

fzf-repl communicates with fzf's HTTP API:

- **GET** `/` — Returns program state as JSON. Params: `limit`, `offset`
- **POST** `/` — Sends actions. Body contains the action string

See `fzf --man` for the complete `--listen` documentation.
