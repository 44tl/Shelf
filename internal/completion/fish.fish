complete -c shelf -f

complete -c shelf -l help -s h -d "show help"
complete -c shelf -l version -s v -d "print the version"
complete -c shelf -l apply -s a -d "perform the moves"
complete -c shelf -l watch -s w -d "keep organizing as files arrive"
complete -c shelf -l undo -d "revert the most recent applied run"
complete -c shelf -l init -d "write a starter rules file to edit"
complete -c shelf -l config -rF -d "use a specific rules file"
complete -c shelf -l completion -x -a "bash zsh fish powershell" -d "print a completion script"
complete -c shelf -kfa '(__fish_complete_directories (commandline -ct))' -d "folder to tidy"
