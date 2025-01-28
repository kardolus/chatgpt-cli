package internal

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

type LevelSet map[zapcore.Level]bool

func (ls LevelSet) Enabled(l zapcore.Level) bool {
	return ls[l]
}

var logLevels LevelSet

func SetAllowedLogLevels(levels ...zapcore.Level) {
	newLevels := make(LevelSet)
	for _, lvl := range levels {
		newLevels[lvl] = true
	}
	logLevels = newLevels
	InitLogger()
}

func InitLogger() {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "", // Disable timestamp
		LevelKey:      "", // Disable log level
		CallerKey:     "", // Disable caller
		FunctionKey:   "", // Disable function name
		StacktraceKey: "", // Disable stacktrace
		MessageKey:    "msg",
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	stdoutWriter := zapcore.Lock(os.Stdout)
	stderrWriter := zapcore.Lock(os.Stderr)

	// INFO & (optionally) DEBUG logs → stdout
	stdoutCore := zapcore.NewCore(consoleEncoder, stdoutWriter, zap.LevelEnablerFunc(logLevels.Enabled))

	// WARN, ERROR, and FATAL logs → stderr (always enabled)
	stderrCore := zapcore.NewCore(consoleEncoder, stderrWriter, zap.LevelEnablerFunc(func(l zapcore.Level) bool {
		return l >= zapcore.WarnLevel
	}))

	logger := zap.New(zapcore.NewTee(stdoutCore, stderrCore))

	zap.ReplaceGlobals(logger)
}
