Register-ArgumentCompleter -Native -CommandName shelf -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    if ($wordToComplete -like '-*') {
        $flags = @('--apply', '--watch', '--undo', '--init',
                   '--config', '--completion', '--help', '--version')
        $flags | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
        }
        return
    }

    Get-ChildItem -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -like "$wordToComplete*" } | ForEach-Object {
            $path = $_.FullName
            [System.Management.Automation.CompletionResult]::new($path, $path, 'ProviderItem', $path)
        }
}
