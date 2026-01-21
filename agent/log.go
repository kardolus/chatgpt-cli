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

	HumanZap *zap.Logger
	DebugZap *zap.Logger

	humanFile *os.File
	debugFile *os.File
}

func (l *Logs) Close() {
	// best-effort; ignore errors
	if l.HumanZap != nil {
		_ = l.HumanZap.Sync()
	}
	if l.DebugZap != nil {
		_ = l.DebugZap.Sync()
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

	// Naming + formats:
	// - transcript: human-readable, line-oriented
	// - debug: JSONL for machines/grep/jq tooling
	humanPath := filepath.Join(dir, "agent.transcript.log")
	debugPath := filepath.Join(dir, "agent.debug.jsonl")

	humanSug, humanZap, humanFile, err := newFileLogger(humanPath, zapcore.InfoLevel, false /* json */)
	if err != nil {
		return nil, err
	}

	debugSug, debugZap, debugFile, err := newFileLogger(debugPath, zapcore.DebugLevel, true /* json */)
	if err != nil {
		_ = humanZap.Sync()
		_ = humanFile.Close()
		return nil, err
	}

	return &Logs{
		Dir:         dir,
		HumanPath:   humanPath,
		DebugPath:   debugPath,
		HumanLogger: humanSug,
		DebugLogger: debugSug,
		HumanZap:    humanZap,
		DebugZap:    debugZap,
		humanFile:   humanFile,
		debugFile:   debugFile,
	}, nil
}

func newFileLogger(path string, level zapcore.Level, json bool) (*zap.SugaredLogger, *zap.Logger, *os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, nil, err
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var enc zapcore.Encoder
	if json {
		enc = zapcore.NewJSONEncoder(encCfg) // JSONL: 1 JSON object per line
	} else {
		enc = zapcore.NewConsoleEncoder(encCfg) // human-readable
	}

	core := zapcore.NewCore(enc, zapcore.AddSync(f), level)
	zl := zap.New(core)
	return zl.Sugar(), zl, f, nil
}
