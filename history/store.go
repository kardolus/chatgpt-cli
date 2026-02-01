package history

import (
	"encoding/json"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/internal"
	"os"
	"path"
	"path/filepath"
)

const jsonExtension = ".json"

type Store interface {
	Read() ([]History, error)
	ReadThread(string) ([]History, error)
	Write([]History) error
	SetThread(string)
	GetThread() string
}

// Ensure FileIO implements the HistoryStore interface
var _ Store = &FileIO{}

type FileIO struct {
	historyDir string
	thread     string
}

func New() (*FileIO, error) {
	_ = migrate()

	dir, err := internal.GetDataHome()
	if err != nil {
		return nil, err
	}

	chatGPTDir, err := internal.GetConfigHome()
	if err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(chatGPTDir)
	if err == nil {
		if fileInfo.IsDir() {
			err = os.MkdirAll(dir, 0o700)
		}
	}

	return &FileIO{
		historyDir: dir,
	}, err
}

func (f *FileIO) GetThread() string {
	return f.thread
}

func (f *FileIO) SetThread(thread string) {
	f.thread = thread
}

func (f *FileIO) WithDirectory(historyDir string) *FileIO {
	f.historyDir = historyDir
	return f
}

func (f *FileIO) Read() ([]History, error) {
	return parseFile(f.getPath(f.thread))
}

func (f *FileIO) ReadThread(thread string) ([]History, error) {
	return parseFile(f.getPath(thread))
}

func (f *FileIO) Write(historyEntries []History) error {
	data, err := json.Marshal(historyEntries)
	if err != nil {
		return err
	}

	return os.WriteFile(f.getPath(f.thread), data, 0o600)
}

func (f *FileIO) getPath(thread string) string {
	return filepath.Join(f.historyDir, thread+jsonExtension)
}

// migrate moves the legacy "history" file in ~/.chatgpt-cli to "history/default.json"
func migrate() error {
	hiddenDir, err := internal.GetConfigHome()
	if err != nil {
		return err
	}

	historyFile, err := internal.GetDataHome()
	if err != nil {
		return err
	}

	fileInfo, err := os.Stat(historyFile)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		defaults := config.NewStore().ReadDefaults()

		// move the legacy "history" file to "default.json"
		if err := os.Rename(historyFile, path.Join(hiddenDir, defaults.Thread+jsonExtension)); err != nil {
			return err
		}

		// create the "history" directory
		if err := os.Mkdir(historyFile, 0755); err != nil {
			return err
		}

		// move default.json to the "history" directory
		if err := os.Rename(path.Join(hiddenDir, defaults.Thread+jsonExtension), path.Join(historyFile, defaults.Thread+jsonExtension)); err != nil {
			return err
		}
	}

	return nil
}

func parseFile(fileName string) ([]History, error) {
	var result []History

	buf, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(buf, &result); err != nil {
		return nil, err
	}

	return result, nil
}
