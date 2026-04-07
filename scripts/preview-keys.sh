#!/bin/bash
# Preview script for fzf: shows keybinding help in which-key style.
# Usage: --preview="scripts/preview-keys.sh {}"
# Can also be run standalone for a quick keybind reference.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Use the fzfg binary if available
FZFG="${FZFG:-$PROJECT_DIR/dist/fzfg}"
if [[ ! -x "$FZFG" ]]; then
  FZFG="$(command -v fzfg 2>/dev/null)"
fi

if [[ -x "$FZFG" ]]; then
  exec "$FZFG" --debug=keybinds
fi

# Fallback: basic display from fzfrc if fzfg not available
echo -e "\033[1;34m  Key Bindings (from fzfrc)\033[0m"
echo -e "\033[90m  ────────────────────────────────────────────────\033[0m"

FZFRC="${FZF_DEFAULT_OPTS_FILE:-$HOME/.fzf/.fzfrc}"
if [[ ! -r "$FZFRC" ]]; then
  FZFRC="$HOME/.fzfrc"
fi

if [[ ! -r "$FZFRC" ]]; then
  echo -e "  \033[90m(no fzfrc found)\033[0m"
  exit 0
fi

desc=""
while IFS= read -r line; do
  trimmed="${line#"${line%%[![:space:]]*}"}"
  # Capture comment
  if [[ "$trimmed" =~ ^#\ (.+) ]]; then
    desc="${BASH_REMATCH[1]}"
    continue
  fi
  # Parse --bind line
  if [[ "$trimmed" =~ ^--bind\ +[\'\"]*([^:]+):(.+)[\'\"]*$ ]]; then
    key="${BASH_REMATCH[1]}"
    action="${BASH_REMATCH[2]}"
    # Truncate long actions
    if [[ ${#action} -gt 35 ]]; then
      action="${action:0:32}..."
    fi
    printf "  \033[1;36m%-16s\033[0m \033[37m%-35s\033[0m \033[90m%s\033[0m\n" \
      "$key" "$action" "$desc"
    desc=""
  elif [[ -n "$trimmed" && ! "$trimmed" =~ ^# ]]; then
    desc=""
  fi
done < "$FZFRC"
