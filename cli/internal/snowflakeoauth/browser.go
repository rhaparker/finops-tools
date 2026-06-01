package snowflakeoauth

import (
	"os/exec"
	"runtime"
)

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		if err := exec.Command("xdg-open", url).Start(); err == nil {
			return nil
		}
		return exec.Command("x-www-browser", url).Start()
	}
}
