#!/bin/bash
# shellcheck disable=SC1090,SC1091

unset FZF_DEFAULT_COMMAND
unset FZF_DEFAULT_OPTS
FZFG_CONF="$PWD/scripts/fzfg.yaml"

if [[ -r $FZFG_CONF ]]; then
  export FZFG_CONF
elif [[ -r "$HOME/.config/fzfg/fzfg.yaml" ]]; then
  export FZFG_CONF="$HOME/.config/fzfg/fzfg.yaml"
elif [[ -r "./configs/fzfg.yaml" ]]; then
  export FZFG_CONF="$PWD/configs/fzfg.yaml"
elif [[ -r "./fzfg.yaml" ]]; then
  export FZFG_CONF="$PWD/fzfg.yaml"
else
  printf "\e[33mUnable to find configuration file\e[0m\n" >&2
  exit 1
fi
printf "\e[32mConfiguration file : %s\e[0m\n" "$FZFG_CONF" >&2

if [[ -r "./shell/completions.bash" ]]; then
  source "./shell/completions.bash"
  printf "\e[32mCompletions file : %s\e[0m\n" "./shell/completions.bash" >&2
elif [[ -r "$HOME/.config/fzfg/completions.bash" ]]; then
  source "$HOME/.config/fzfg/completions.bash"
  printf "\e[32mCompletions file : %s\e[0m\n" "$HOME/.config/fzfg/completions.bash" >&2
else
  printf "\e[33mUnable to find completions file\e[0m\n" >&2
  exit 1
fi
