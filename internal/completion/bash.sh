# bash completion for shelf
_shelf() {
  local cur prev
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  case "$prev" in
    --completion)
      COMPREPLY=($(compgen -W "bash zsh fish powershell" -- "$cur"))
      return
      ;;
    --config)
      compopt -o filenames 2>/dev/null
      COMPREPLY=($(compgen -f -- "$cur"))
      return
      ;;
  esac

  if [[ "$cur" == -* ]]; then
    COMPREPLY=($(compgen -W "--apply --watch --undo --init --config --completion --help --version" -- "$cur"))
    return
  fi

  compopt -o filenames 2>/dev/null
  COMPREPLY=($(compgen -d -- "$cur"))
}
complete -F _shelf shelf
