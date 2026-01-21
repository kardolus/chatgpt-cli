package tools

import (
	"bytes"
	"fmt"
	"github.com/kardolus/chatgpt-cli/agent/utils"

	"github.com/kardolus/chatgpt-cli/internal/fsio"
)

type PatchResult struct {
	Hunks int
}

type ReplaceResult struct {
	OccurrencesFound int
	Replaced         int
}

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	PatchFile(path string, unifiedDiff []byte) (PatchResult, error)
	ReplaceBytesInFile(path string, old, new []byte, n int) (ReplaceResult, error)
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
	defer func() { _ = file.Close() }()

	if err := f.w.Write(file, data); err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}

	return nil
}

func (f FSIOFileOps) PatchFile(path string, unifiedDiff []byte) (PatchResult, error) {
	// parse once for stats (and early validation)
	hunks, err := utils.ParseUnifiedDiff(unifiedDiff)
	if err != nil {
		return PatchResult{}, err
	}

	orig, err := f.ReadFile(path)
	if err != nil {
		return PatchResult{}, err
	}

	patched, err := utils.ApplyUnifiedDiff(orig, unifiedDiff)
	if err != nil {
		return PatchResult{Hunks: len(hunks)}, fmt.Errorf("apply patch %s: %w", path, err)
	}

	if bytes.Equal(orig, patched) {
		return PatchResult{Hunks: len(hunks)}, nil
	}

	if err := f.WriteFile(path, patched); err != nil {
		return PatchResult{Hunks: len(hunks)}, err
	}

	return PatchResult{Hunks: len(hunks)}, nil
}

func (f FSIOFileOps) ReplaceBytesInFile(path string, old, new []byte, n int) (ReplaceResult, error) {
	if len(old) == 0 {
		return ReplaceResult{}, fmt.Errorf("replace %s: old pattern must be non-empty", path)
	}

	orig, err := f.ReadFile(path)
	if err != nil {
		return ReplaceResult{}, err
	}

	found := bytes.Count(orig, old)
	if found == 0 {
		return ReplaceResult{OccurrencesFound: 0, Replaced: 0}, fmt.Errorf("replace %s: pattern not found", path)
	}

	limit := n
	if n <= 0 {
		limit = -1 // replace all
	}

	updated := bytes.Replace(orig, old, new, limit)
	if bytes.Equal(orig, updated) {
		return ReplaceResult{OccurrencesFound: found, Replaced: 0}, fmt.Errorf("replace %s: no changes applied", path)
	}

	if err := f.WriteFile(path, updated); err != nil {
		return ReplaceResult{OccurrencesFound: found, Replaced: 0}, err
	}

	replaced := found
	if n > 0 && n < found {
		replaced = n
	}

	return ReplaceResult{OccurrencesFound: found, Replaced: replaced}, nil
}
