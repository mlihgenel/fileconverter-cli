package installer

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// InstallInfo kurulum bilgisini tutar
type InstallInfo struct {
	ToolName    string
	Command     string
	Args        []string
	Description string
	ManualURL   string
	Supported   bool // Otomatik kurulum destekleniyor mu
}

// DetectPackageManager mevcut paket yöneticisini tespit eder
func DetectPackageManager() string {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("brew"); err == nil {
			return "brew"
		}
		return ""
	case "linux":
		// apt (Debian/Ubuntu)
		if _, err := exec.LookPath("apt"); err == nil {
			return "apt"
		}
		// dnf (Fedora)
		if _, err := exec.LookPath("dnf"); err == nil {
			return "dnf"
		}
		// yum (CentOS/RHEL)
		if _, err := exec.LookPath("yum"); err == nil {
			return "yum"
		}
		// pacman (Arch)
		if _, err := exec.LookPath("pacman"); err == nil {
			return "pacman"
		}
		return ""
	case "windows":
		// Chocolatey
		if _, err := exec.LookPath("choco"); err == nil {
			return "choco"
		}
		// Winget
		if _, err := exec.LookPath("winget"); err == nil {
			return "winget"
		}
		return ""
	}
	return ""
}

// GetInstallInfo belirli bir araç için kurulum bilgilerini döner
func GetInstallInfo(toolName string) InstallInfo {
	pm := DetectPackageManager()
	toolName = strings.ToLower(toolName)

	switch toolName {
	case "ffmpeg":
		return getFFmpegInstall(pm)
	case "pandoc":
		return getPandocInstall(pm)
	case "libreoffice":
		return getLibreOfficeInstall(pm)
	}

	return InstallInfo{
		ToolName:  toolName,
		Supported: false,
		ManualURL: "",
	}
}

func getFFmpegInstall(pm string) InstallInfo {
	info := InstallInfo{
		ToolName:  "FFmpeg",
		ManualURL: "https://ffmpeg.org/download.html",
	}

	switch pm {
	case "brew":
		info.Command = "brew"
		info.Args = []string{"install", "ffmpeg"}
		info.Description = "brew install ffmpeg"
		info.Supported = true
	case "apt":
		info.Command = "sudo"
		info.Args = []string{"apt", "install", "-y", "ffmpeg"}
		info.Description = "sudo apt install -y ffmpeg"
		info.Supported = true
	case "dnf":
		info.Command = "sudo"
		info.Args = []string{"dnf", "install", "-y", "ffmpeg"}
		info.Description = "sudo dnf install -y ffmpeg"
		info.Supported = true
	case "yum":
		info.Command = "sudo"
		info.Args = []string{"yum", "install", "-y", "ffmpeg"}
		info.Description = "sudo yum install -y ffmpeg"
		info.Supported = true
	case "pacman":
		info.Command = "sudo"
		info.Args = []string{"pacman", "-S", "--noconfirm", "ffmpeg"}
		info.Description = "sudo pacman -S --noconfirm ffmpeg"
		info.Supported = true
	case "choco":
		info.Command = "choco"
		info.Args = []string{"install", "ffmpeg", "-y"}
		info.Description = "choco install ffmpeg -y"
		info.Supported = true
	case "winget":
		info.Command = "winget"
		info.Args = []string{"install", "Gyan.FFmpeg"}
		info.Description = "winget install Gyan.FFmpeg"
		info.Supported = true
	default:
		info.Supported = false
	}

	return info
}

func getPandocInstall(pm string) InstallInfo {
	info := InstallInfo{
		ToolName:  "Pandoc",
		ManualURL: "https://pandoc.org/installing.html",
	}

	switch pm {
	case "brew":
		info.Command = "brew"
		info.Args = []string{"install", "pandoc"}
		info.Description = "brew install pandoc"
		info.Supported = true
	case "apt":
		info.Command = "sudo"
		info.Args = []string{"apt", "install", "-y", "pandoc"}
		info.Description = "sudo apt install -y pandoc"
		info.Supported = true
	case "dnf":
		info.Command = "sudo"
		info.Args = []string{"dnf", "install", "-y", "pandoc"}
		info.Description = "sudo dnf install -y pandoc"
		info.Supported = true
	case "yum":
		info.Command = "sudo"
		info.Args = []string{"yum", "install", "-y", "pandoc"}
		info.Description = "sudo yum install -y pandoc"
		info.Supported = true
	case "pacman":
		info.Command = "sudo"
		info.Args = []string{"pacman", "-S", "--noconfirm", "pandoc"}
		info.Description = "sudo pacman -S --noconfirm pandoc"
		info.Supported = true
	case "choco":
		info.Command = "choco"
		info.Args = []string{"install", "pandoc", "-y"}
		info.Description = "choco install pandoc -y"
		info.Supported = true
	case "winget":
		info.Command = "winget"
		info.Args = []string{"install", "JohnMacFarlane.Pandoc"}
		info.Description = "winget install JohnMacFarlane.Pandoc"
		info.Supported = true
	default:
		info.Supported = false
	}

	return info
}

func getLibreOfficeInstall(pm string) InstallInfo {
	info := InstallInfo{
		ToolName:  "LibreOffice",
		ManualURL: "https://www.libreoffice.org/download",
	}

	switch pm {
	case "brew":
		info.Command = "brew"
		info.Args = []string{"install", "--cask", "libreoffice"}
		info.Description = "brew install --cask libreoffice"
		info.Supported = true
	case "apt":
		info.Command = "sudo"
		info.Args = []string{"apt", "install", "-y", "libreoffice"}
		info.Description = "sudo apt install -y libreoffice"
		info.Supported = true
	case "dnf":
		info.Command = "sudo"
		info.Args = []string{"dnf", "install", "-y", "libreoffice"}
		info.Description = "sudo dnf install -y libreoffice"
		info.Supported = true
	case "yum":
		info.Command = "sudo"
		info.Args = []string{"yum", "install", "-y", "libreoffice"}
		info.Description = "sudo yum install -y libreoffice"
		info.Supported = true
	case "pacman":
		info.Command = "sudo"
		info.Args = []string{"pacman", "-S", "--noconfirm", "libreoffice-fresh"}
		info.Description = "sudo pacman -S --noconfirm libreoffice-fresh"
		info.Supported = true
	case "choco":
		info.Command = "choco"
		info.Args = []string{"install", "libreoffice-fresh", "-y"}
		info.Description = "choco install libreoffice-fresh -y"
		info.Supported = true
	case "winget":
		info.Command = "winget"
		info.Args = []string{"install", "TheDocumentFoundation.LibreOffice"}
		info.Description = "winget install TheDocumentFoundation.LibreOffice"
		info.Supported = true
	default:
		info.Supported = false
	}

	return info
}

// InstallTool belirli bir aracı kurar
func InstallTool(toolName string) (string, error) {
	info := GetInstallInfo(toolName)

	if !info.Supported {
		return "", fmt.Errorf(
			"%s otomatik olarak kurulamıyor.\nManuel kurulum: %s",
			info.ToolName, info.ManualURL,
		)
	}

	cmd := exec.Command(info.Command, info.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s kurulumu başarısız: %w", info.ToolName, err)
	}

	return info.Description, nil
}

// GetMissingToolNames eksik araçların isimlerini döner
func GetMissingToolNames(tools []string) []string {
	var missing []string
	for _, tool := range tools {
		t := strings.ToLower(tool)
		switch t {
		case "ffmpeg":
			if _, err := exec.LookPath("ffmpeg"); err != nil {
				missing = append(missing, tool)
			}
		case "pandoc":
			if _, err := exec.LookPath("pandoc"); err != nil {
				missing = append(missing, tool)
			}
		case "libreoffice":
			// macOS check
			paths := []string{"soffice", "libreoffice"}
			found := false
			for _, p := range paths {
				if _, err := exec.LookPath(p); err == nil {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, tool)
			}
		}
	}
	return missing
}
