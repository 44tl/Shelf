#compdef shelf
compdef _shelf shelf

_shelf() {
  _arguments -S \
    '(--help)-h[show help]' \
    '(--version)-v[print the version]' \
    '--apply[perform the moves]' \
    '--watch[keep organizing as files arrive]' \
    '--undo[revert the most recent applied run]' \
    '--init[write a starter rules file to edit]' \
    '--config[use a specific rules file]:rules file:_files' \
    '--completion[print a completion script]:shell:(bash zsh fish powershell)' \
    '*:folder to tidy:_directories'
}
