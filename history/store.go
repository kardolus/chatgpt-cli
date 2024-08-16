package history

import (
	"encoding/json"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/kardolus/chatgpt-cli/utils"
	"os"
	"path"
	"path/filepath"
)

const (
	jsonExtension = ".json"
)

type HistoryStore interface {
	Read() ([]types.Message, error)
	Write([]types.Message) error
	SetThread(thread string)
	GetThread() string
}

// Ensure FileIO implements the HistoryStore interface
var _ HistoryStore = &FileIO{}

type FileIO struct {
	historyDir string
	thread     string
}

func New() (*FileIO, error) {
	_ = migrate()

	dir, err := utils.GetHistoryDir()
	if err != nil {
		return nil, err
	}

	chatGPTDir, err := utils.GetChatGPTDirectory()
	if err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(chatGPTDir)
	if err == nil {
		if fileInfo.IsDir() {
			err = os.MkdirAll(dir, 0755)
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

func (f *FileIO) Read() ([]types.Message, error) {
	return parseFile(f.getPath())
}

func (f *FileIO) Write(messages []types.Message) error {
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	return os.WriteFile(f.getPath(), data, 0644)
}

func (f *FileIO) getPath() string {
	return filepath.Join(f.historyDir, f.thread+jsonExtension)
}

// migrate moves the legacy "history" file in ~/.chatgpt-cli to "history/default.json"
func migrate() error {
	hiddenDir, err := utils.GetChatGPTDirectory()
	if err != nil {
		return err
	}

	historyFile, err := utils.GetHistoryDir()
	if err != nil {
		return err
	}

	fileInfo, err := os.Stat(historyFile)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		defaults := config.New().ReadDefaults()

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

func parseFile(fileName string) ([]types.Message, error) {
	var result []types.Message

	buf, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(buf, &result); err != nil {
		return nil, err
	}

	return result, nil
}
