package sourcecode

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	lru "github.com/hashicorp/golang-lru/v2"
)

type contentManager struct {
	contentCache *lru.Cache[string, []string]
}

func newContentManager() contentManager {
	contentCache, err := lru.New[string, []string](5)
	if err != nil {
		panic(err)
	}
	return contentManager{
		contentCache: contentCache,
	}
}

func (m *contentManager) getSourceCode(filename string) ([]string, error) {
	if content, ok := m.contentCache.Get(filename); ok {
		return content, nil
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error getting current file content: error opening file: %s: %v", filename, err)
	}

	colorizedContent, err := m.colorize(string(content))
	if err != nil {
		return nil, fmt.Errorf("error highlighting the source code: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(colorizedContent), "\n")
	m.contentCache.Add(filename, lines)

	return lines, nil
}

func (m *contentManager) colorize(content string) (string, error) {
	sb := strings.Builder{}

	err := quick.Highlight(&sb, content, "go", "terminal8", "native")
	if err != nil {
		return "", fmt.Errorf("error highlighting the source code: %w", err)
	}

	return sb.String(), nil
}
