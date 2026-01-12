package agent

import (
	"fmt"

	"github.com/kardolus/chatgpt-cli/internal/fsio"
)

//go:generate mockgen -destination=filemocks_test.go -package=agent_test github.com/kardolus/chatgpt-cli/agent Files
type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
}

type FSIOFileOps struct {
	r fsio.Reader
	w fsio.Writer
}

func NewFSIOFileOps(r fsio.Reader, w fsio.Writer) FSIOFileOps {
	return FSIOFileOps{r: r, w: w}
}

func (f FSIOFileOps) ReadFile(path string) ([]byte, error) {
	return f.r.ReadFile(path)
}

func (f FSIOFileOps) WriteFile(path string, data []byte) error {
	file, err := f.w.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		_ = file.Close()
	}()

	if err := f.w.Write(file, data); err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}

	return nil
}
