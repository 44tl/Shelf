package completion

import (
	"embed"
	"fmt"
)

//go:embed bash.sh zsh.sh fish.fish powershell.ps1
var scripts embed.FS

func Script(shell string) (string, error) {
	var name string
	switch shell {
	case "bash":
		name = "bash.sh"
	case "zsh":
		name = "zsh.sh"
	case "fish":
		name = "fish.fish"
	case "powershell", "pwsh":
		name = "powershell.ps1"
	default:
		return "", fmt.Errorf("unknown shell %q — try bash, zsh, fish or powershell", shell)
	}
	b, err := scripts.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Shells() []string {
	return []string{"bash", "zsh", "fish", "powershell"}
}
