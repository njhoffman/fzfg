# shellcheck shell=bash

__fzfg_config_list_subkeys() {
  local key_rx='^([^#: ]+):' subkey_rx='^  ([^#: ]+):' key
  while IFS= read -r line; do
    [[ $line =~ $key_rx ]] && key="${BASH_REMATCH[1]}"
    [[ $line =~ $subkey_rx && $key == "$1" ]] && echo "${BASH_REMATCH[1]}"
  done < "$FZFG_CONF"
}

__fzfg_config_match_subkeys() {
  __fzfg_config_list_subkeys "$1" | while IFS= read -r line; do
    [[ -z $2 || (-n $2 && $line == *"$2"*) ]] && echo "${line//[$' \t']/}"
  done
}

_fzfg_config_compgen() {
  local curr="$1" prev="$2"
  case "$prev" in
    -c)
      __fzfg_config_match_subkeys commands "$curr"
      ;;
    -o)
      __fzfg_config_match_subkeys options "$curr"
      ;;
    -p)
      __fzfg_config_match_subkeys profiles "$curr"
      ;;
    *)
      echo "-c -o -p"
      ;;
  esac
}

_fzfg_config_complete() {
  local curr prev words
  COMPREPLY=()
  curr="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  words="$(_fzfg_config_compgen "$curr" "$prev")"
  [[ -z $words ]] && return 1

  read -rd '' -a COMPREPLY < <(compgen -W "$words" -G "$curr")
  return 0
}

complete -F _fzfg_config_complete fzfg
