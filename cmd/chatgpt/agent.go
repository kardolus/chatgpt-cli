package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kardolus/chatgpt-cli/agent/core"
	"github.com/kardolus/chatgpt-cli/agent/factory"
	"github.com/kardolus/chatgpt-cli/agent/planexec"
	"github.com/kardolus/chatgpt-cli/agent/tools"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/cmd/chatgpt/utils"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/internal/fsio"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func appendAgentRunToHistory(store history.Store, systemRole, goal, answer string) error {
	thread := store.GetThread()

	entries, err := store.ReadThread(thread)
	if err != nil {
		// treat missing history file as empty thread
		if errors.Is(err, os.ErrNotExist) {
			entries = nil
		} else {
			return err
		}
	}

	now := time.Now()

	// Ensure a system message exists at the beginning (optional but matches your UX)
	if len(entries) == 0 || strings.ToLower(strings.TrimSpace(entries[0].Role)) != "system" {
		entries = append(entries, history.History{
			Message: api.Message{
				Role:    "system",
				Content: systemRole,
			},
			Timestamp: now,
		})
	}

	entries = append(entries,
		history.History{
			Message:   api.Message{Role: "user", Content: goal},
			Timestamp: now,
		},
		history.History{
			Message:   api.Message{Role: "assistant", Content: answer},
			Timestamp: now,
		},
	)

	// store already has thread set, but being explicit is fine
	store.SetThread(thread)
	return store.Write(entries)
}

func runAgent(ctx context.Context, c *client.Client, cfg config.Config, mode string, goal string) (string, error) {
	clk := core.NewRealClock()
	llm := tools.NewClientLLM(c)

	tools, err := buildAgentTools(llm)
	if err != nil {
		return "", err
	}

	policy, err := buildAgentPolicy(cfg)
	if err != nil {
		return "", err
	}

	budget := core.NewDefaultBudget(utils.BudgetLimitsFromConfig(cfg))
	runner := core.NewDefaultRunner(tools, clk, budget, policy)

	logs, err := core.NewLogs()
	if err != nil {
		return "", err
	}
	defer logs.Close()

	// Tee human output: terminal + transcript file.
	// zap.L() uses the global logger core (terminal), logs.HumanZap.Core() writes to transcript file.
	humanTeeZap := zap.New(zapcore.NewTee(
		zap.L().Core(),
		logs.HumanZap.Core(),
	))
	humanTeeSug := humanTeeZap.Sugar()

	baseOpts := []core.BaseOption{
		core.WithWorkDir(cfg.Agent.WorkDir),
		core.WithDryRun(cfg.Agent.DryRun),

		// Human (transcript): terminal + file
		core.WithHumanLogger(humanTeeSug, func() {
			// best-effort sync
			_ = humanTeeZap.Sync()
		}),

		// Debug: JSONL file only (already JSON encoder in logs.go)
		core.WithDebugLogger(logs.DebugLogger, func() {
			_ = logs.DebugZap.Sync()
		}),
	}

	switch mode {
	case "react":
		a, err := factory.New(factory.ModeReAct, factory.Deps{
			Clock:  clk,
			LLM:    llm,
			Runner: runner,
			Budget: budget,
		}, baseOpts...)
		if err != nil {
			return "", err
		}
		return a.RunAgentGoal(ctx, goal)

	case "plan":
		var planner planexec.Planner = planexec.NewDefaultPlanner(
			llm,
			budget,
			clk,
			planexec.WithPlannerRawSink(func(raw string) {
				if !cfg.Agent.WritePlanJSON {
					return
				}
				planPath := strings.TrimSpace(cfg.Agent.PlanJSONPath)
				if planPath == "" {
					planPath = filepath.Join(logs.Dir, "plan.json")
				}
				_ = os.WriteFile(planPath, []byte(strings.TrimSpace(raw)), 0o644)
			}),
		)

		planner = planexec.NewLoggingPlanner(planner, logs)

		a, err := factory.New(factory.ModePlanExecute, factory.Deps{
			Clock:   clk,
			Planner: planner,
			Runner:  runner,
			LLM:     llm,
			Budget:  budget,
		}, baseOpts...)
		if err != nil {
			return "", err
		}
		return a.RunAgentGoal(ctx, goal)

	default:
		return "", fmt.Errorf("internal error: unsupported mode %q", mode)
	}
}

func buildAgentTools(llm tools.LLM) (core.Tools, error) {
	sh := tools.NewExecShellRunner()
	r := fsio.NewRealReader(fsio.DefaultBufferSize)
	w := &fsio.RealWriter{}
	files := tools.NewFSIOFileOps(r, w)

	return core.Tools{
		Shell: sh,
		LLM:   llm,
		Files: files,
	}, nil
}

func buildAgentPolicy(cfg config.Config) (core.Policy, error) {
	allowedTools, err := utils.ParseToolKinds(cfg.Agent.AllowedTools)
	if err != nil {
		return nil, err
	}

	return core.NewDefaultPolicy(core.PolicyLimits{
		AllowedTools:           allowedTools,
		DeniedShellCommands:    cfg.Agent.DeniedShellCommands,
		AllowedFileOps:         cfg.Agent.AllowedFileOps,
		RestrictFilesToWorkDir: cfg.Agent.RestrictFilesToWorkDir,
	}), nil
}
