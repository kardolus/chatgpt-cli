package agent

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/kardolus/chatgpt-cli/internal"
)

type Logs struct {
	Dir       string
	HumanPath string
	DebugPath string

	HumanLogger *zap.SugaredLogger
	DebugLogger *zap.SugaredLogger

	humanZap *zap.Logger
	debugZap *zap.Logger

	humanFile *os.File
	debugFile *os.File
}

func (l *Logs) Close() {
	// best-effort; ignore errors
	if l.HumanLogger != nil && l.humanZap != nil {
		_ = l.humanZap.Sync()
	}
	if l.DebugLogger != nil && l.debugZap != nil {
		_ = l.debugZap.Sync()
	}
	if l.humanFile != nil {
		_ = l.humanFile.Close()
	}
	if l.debugFile != nil {
		_ = l.debugFile.Close()
	}
}

func NewLogs() (*Logs, error) {
	cacheHome, err := internal.GetCacheHome()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(cacheHome, "agent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	humanPath := filepath.Join(dir, "agent.log")
	debugPath := filepath.Join(dir, "agent.debug.log")

	humanSug, humanZap, humanFile, err := newFileLogger(humanPath, zapcore.InfoLevel)
	if err != nil {
		return nil, err
	}

	debugSug, debugZap, debugFile, err := newFileLogger(debugPath, zapcore.DebugLevel)
	if err != nil {
		return nil, err
	}

	return &Logs{
		Dir:         dir,
		HumanPath:   humanPath,
		DebugPath:   debugPath,
		HumanLogger: humanSug,
		DebugLogger: debugSug,
		humanZap:    humanZap,
		debugZap:    debugZap,
		humanFile:   humanFile,
		debugFile:   debugFile,
	}, nil
}

func newFileLogger(path string, level zapcore.Level) (*zap.SugaredLogger, *zap.Logger, *os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, nil, err
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	enc := zapcore.NewConsoleEncoder(encCfg)

	ws := zapcore.AddSync(f)
	core := zapcore.NewCore(enc, ws, level)

	zl := zap.New(core)
	return zl.Sugar(), zl, f, nil
}
