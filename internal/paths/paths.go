package paths

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func GetProjectRoot() string {
	goModPath, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return ""
	}
	if len(goModPath) == 0 {
		return ""
	}

	return filepath.Dir(strings.TrimSpace(string(goModPath)))
}

func Trunc(path string, maxWidth int) string {
	if len(path) <= maxWidth {
		return path
	}

	dir, filename := filepath.Split(path)
	if len(filename) >= maxWidth {
		return filename
	}

	availableSpace := maxWidth - len(filename) - 3
	if availableSpace <= 0 {
		return filename
	}

	dirParts := strings.Split(strings.TrimSuffix(dir, string(filepath.Separator)), string(filepath.Separator))

	var truncatedDir string

	for i := len(dirParts) - 1; i >= 0; i-- {
		nextPart := dirParts[i]
		if i < len(dirParts)-1 {
			nextPart += string(filepath.Separator)
		}

		if len(nextPart)+len(truncatedDir) > availableSpace {
			if truncatedDir != "" {
				break
			}
			if len(nextPart) > availableSpace {
				truncatedDir = nextPart[len(nextPart)-availableSpace:]
			} else {
				truncatedDir = nextPart
			}
			break
		}

		truncatedDir = nextPart + truncatedDir
	}

	if truncatedDir != "" && !strings.HasSuffix(truncatedDir, string(filepath.Separator)) {
		truncatedDir += string(filepath.Separator)
	}

	return "..." + truncatedDir + filename
}
