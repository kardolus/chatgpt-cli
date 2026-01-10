package utils_test

import (
	"fmt"
	"github.com/kardolus/chatgpt-cli/cmd/chatgpt/utils"
	"github.com/kardolus/chatgpt-cli/config"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitUtils(t *testing.T) {
	spec.Run(t, "Testing the Utils", testUtils, spec.Report(report.Terminal{}))
}

func testUtils(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("ColorToAnsi()", func() {
		it("should return an empty color and reset if the input is an empty string", func() {
			color, reset := utils.ColorToAnsi("")
			Expect(color).To(Equal(""))
			Expect(reset).To(Equal(""))
		})

		it("should return an empty color and reset if the input is an unsupported color", func() {
			color, reset := utils.ColorToAnsi("unsupported")
			Expect(color).To(Equal(""))
			Expect(reset).To(Equal(""))
		})

		it("should return the correct ANSI code for red", func() {
			color, reset := utils.ColorToAnsi("red")
			Expect(color).To(Equal("\033[31m"))
			Expect(reset).To(Equal("\033[0m"))
		})

		it("should return the correct ANSI code for green", func() {
			color, reset := utils.ColorToAnsi("green")
			Expect(color).To(Equal("\033[32m"))
			Expect(reset).To(Equal("\033[0m"))
		})

		it("should return the correct ANSI code for yellow", func() {
			color, reset := utils.ColorToAnsi("yellow")
			Expect(color).To(Equal("\033[33m"))
			Expect(reset).To(Equal("\033[0m"))
		})

		it("should return the correct ANSI code for blue", func() {
			color, reset := utils.ColorToAnsi("blue")
			Expect(color).To(Equal("\033[34m"))
			Expect(reset).To(Equal("\033[0m"))
		})

		it("should return the correct ANSI code for magenta", func() {
			color, reset := utils.ColorToAnsi("magenta")
			Expect(color).To(Equal("\033[35m"))
			Expect(reset).To(Equal("\033[0m"))
		})

		it("should handle case-insensitivity correctly", func() {
			color, reset := utils.ColorToAnsi("ReD")
			Expect(color).To(Equal("\033[31m"))
			Expect(reset).To(Equal("\033[0m"))
		})

		it("should handle leading and trailing spaces", func() {
			color, reset := utils.ColorToAnsi("  blue ")
			Expect(color).To(Equal("\033[34m"))
			Expect(reset).To(Equal("\033[0m"))
		})
	})

	when("FormatPrompt()", func() {
		const (
			counter = 1
			usage   = 2
		)

		now := time.Now()

		it("should add a trailing whitespace", func() {
			input := "prompt"
			expected := "prompt "
			Expect(utils.FormatPrompt(input, counter, usage, now)).To(Equal(expected))
		})

		it("should handle empty input as expected", func() {
			input := ""
			expected := ""
			Expect(utils.FormatPrompt(input, counter, usage, now)).To(Equal(expected))
		})

		it("should replace %date with the current date", func() {
			currentDate := now.Format("2006-01-02")
			input := "Today's date is %date"
			expected := "Today's date is " + currentDate + " "
			Expect(utils.FormatPrompt(input, counter, usage, now)).To(Equal(expected))
		})

		it("should replace %time with the current time", func() {
			currentTime := now.Format("15:04:05")
			input := "Current time is %time"
			expected := "Current time is " + currentTime + " "
			Expect(utils.FormatPrompt(input, counter, usage, now)).To(Equal(expected))
		})

		it("should replace %datetime with the current date and time", func() {
			currentDatetime := now.Format("2006-01-02 15:04:05")
			input := "Current date and time is %datetime"
			expected := "Current date and time is " + currentDatetime + " "
			Expect(utils.FormatPrompt(input, counter, usage, now)).To(Equal(expected))
		})

		it("should replace %counter with the current counter value", func() {
			input := "The counter is %counter"
			expected := "The counter is 1 "
			Expect(utils.FormatPrompt(input, counter, usage, now)).To(Equal(expected))
		})

		it("should replace %usage with the current usage value", func() {
			input := "The usage is %usage"
			expected := "The usage is 2 "
			Expect(utils.FormatPrompt(input, counter, usage, now)).To(Equal(expected))
		})

		it("should handle complex cases correctly", func() {
			input := "command_prompt: [%time] [Q%counter]"
			expected := fmt.Sprintf("command_prompt: [%s] [Q%d] ", now.Format("15:04:05"), counter)
			Expect(utils.FormatPrompt(input, counter, usage, now)).To(Equal(expected))
		})

		it("should replace \\n with an actual newline", func() {
			input := "Line 1\\nLine 2"
			expected := "Line 1\nLine 2 "
			Expect(utils.FormatPrompt(input, counter, usage, now)).To(Equal(expected))
		})
	})

	when("IsBinary()", func() {
		it("should return false for a regular string", func() {
			Expect(utils.IsBinary([]byte("regular string"))).To(BeFalse())
		})
		it("should return false for a string containing emojis", func() {
			Expect(utils.IsBinary([]byte("☮️✅❤️"))).To(BeFalse())
		})
		it("should return true for a binary string", func() {
			Expect(utils.IsBinary([]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB})).To(BeTrue())
		})
		it("should return false when the data is empty", func() {
			Expect(utils.IsBinary([]byte{})).To(BeFalse())
		})
		it("should handle large text files correctly", func() {
			// Create a large slice > 512KB with normal text
			largeText := make([]byte, 1024*1024) // 1MB
			for i := range largeText {
				largeText[i] = 'a'
			}

			Expect(utils.IsBinary(largeText)).To(BeFalse())
		})
		it("should return true when data contains null bytes", func() {
			Expect(utils.IsBinary([]byte{'h', 'e', 'l', 'l', 0x00, 'o'})).To(BeTrue())
		})

		it("should return true for invalid UTF-8 sequences", func() {
			// Invalid UTF-8: 0xED 0xA0 0x80 is a surrogate pair which is invalid in UTF-8
			Expect(utils.IsBinary([]byte{0xED, 0xA0, 0x80})).To(BeTrue())
		})

		it("should return false for valid UTF-8 special characters", func() {
			// Testing with Chinese characters, Arabic, and other non-ASCII but valid UTF-8
			Expect(utils.IsBinary([]byte("你好世界مرحبا"))).To(BeFalse())
		})

		it("should handle control characters correctly", func() {
			// Test with allowed control characters (tab, newline, carriage return)
			Expect(utils.IsBinary([]byte("Hello\tWorld\r\nTest"))).To(BeFalse())

			// Test with other control characters that should trigger binary detection
			data := []byte{0x01, 0x02, 0x03, 0x04}
			Expect(utils.IsBinary(data)).To(BeTrue())
		})

	})

	when("ValidateFlags()", func() {
		const defaultModel = "mock-model"

		var flags map[string]bool

		it.Before(func() {
			flags = make(map[string]bool)
		})

		it("doesn't throw an error when no flags are provided", func() {
			Expect(utils.ValidateFlags(defaultModel, flags)).To(Succeed())
		})
		it("should return an error when --new-thread and --set-thread are both used", func() {
			flags["new-thread"] = true
			flags["set-thread"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should return an error when --new-thread and --thread are both used", func() {
			flags["new-thread"] = true
			flags["thread"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should return an error when --speak is used but --output is omitted", func() {
			flags["speak"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should return an error when --draw is used but --output is omitted", func() {
			flags["draw"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should return an error when --output is used but --speak or --draw are omitted", func() {
			flags["output"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should return an error when --audio is used with an incompatible model", func() {
			flags["audio"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should NOT return an error when --audio is used with a compatible model", func() {
			flags["audio"] = true

			err := utils.ValidateFlags(defaultModel+utils.AudioPattern, flags)
			Expect(err).NotTo(HaveOccurred())
		})
		it("should return an error when --transcribe is used with an incompatible model", func() {
			flags["transcribe"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should NOT return an error when --transcribe is used with a compatible model", func() {
			flags["transcribe"] = true

			err := utils.ValidateFlags(defaultModel+utils.TranscribePattern, flags)
			Expect(err).NotTo(HaveOccurred())
		})
		it("should return an error when --speak and --output flags are used with an incompatible model", func() {
			flags["speak"] = true
			flags["output"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should NOT return an error when --speak and --output flags are used with a compatible model", func() {
			flags["speak"] = true
			flags["output"] = true

			err := utils.ValidateFlags(defaultModel+utils.TTSPattern, flags)
			Expect(err).NotTo(HaveOccurred())
		})
		it("should return an error when --draw and --output flags are used with an incompatible model", func() {
			flags["draw"] = true
			flags["output"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should NOT return an error when --draw and --output flags are used with a compatible model", func() {
			flags["draw"] = true
			flags["output"] = true

			err := utils.ValidateFlags(defaultModel+utils.ImagePattern, flags)
			Expect(err).NotTo(HaveOccurred())
		})
		it("should NOT return an error when --draw, --image and --output flags are used with a compatible model", func() {
			flags["draw"] = true
			flags["output"] = true
			flags["image"] = true

			err := utils.ValidateFlags(defaultModel+utils.ImagePattern, flags)
			Expect(err).NotTo(HaveOccurred())
		})
		it("should return an error when --voice is used with an incompatible model", func() {
			flags["voice"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should NOT return an error when --voice is used with a compatible model", func() {
			flags["voice"] = true

			err := utils.ValidateFlags(defaultModel+utils.TTSPattern, flags)
			Expect(err).NotTo(HaveOccurred())
		})
		it("should return an error when --effort is used with an incompatible model", func() {
			flags["effort"] = true

			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should NOT return an error when --effort is used with a compatible model", func() {
			flags["effort"] = true

			Expect(utils.ValidateFlags(defaultModel+utils.O1ProPattern, flags)).To(Succeed())
			Expect(utils.ValidateFlags(defaultModel+utils.GPT5Pattern, flags)).To(Succeed())
		})
		it("should return an error when the --mcp-param flag is used without --mcp", func() {
			flags["mcp-param"] = true
			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
		it("should return an error when the --mcp-params flag is used without --mcp", func() {
			flags["mcp-params"] = true
			err := utils.ValidateFlags(defaultModel, flags)
			Expect(err).To(HaveOccurred())
		})
	})

	when("GenerateThreadName()", func() {
		const (
			doNotCreateNewThread    = false
			doNotUseInteractiveMode = false
			useInteractiveMode      = true
			createNewThread         = true
			threadName              = "threadName"
		)
		var cfg config.Config

		it.Before(func() {
			cfg = config.Config{Thread: threadName}
		})

		it("returns the configured thread name when new-thread and auto-new-thread are disabled (regardless of interactive mode)", func() {
			result, updateConfig := utils.GenerateThreadName(cfg, doNotUseInteractiveMode, doNotCreateNewThread)
			Expect(result).To(Equal(threadName))
			Expect(updateConfig).To(BeFalse())

			result, updateConfig = utils.GenerateThreadName(cfg, useInteractiveMode, doNotCreateNewThread)
			Expect(result).To(Equal(threadName))
			Expect(updateConfig).To(BeFalse())
		})

		it("prioritizes --new-thread over auto-create-new-thread when interactive mode is enabled", func() {
			result, updateConfig := utils.GenerateThreadName(cfg, useInteractiveMode, createNewThread)
			Expect(result).To(HavePrefix(utils.CommandPrefix))
			Expect(updateConfig).To(BeTrue())
		})

		it("generates an interactive prefix slug when auto-create-new-thread is enabled", func() {
			cfg.AutoCreateNewThread = createNewThread
			result, updateConfig := utils.GenerateThreadName(cfg, useInteractiveMode, doNotCreateNewThread)
			Expect(result).To(HavePrefix(utils.InteractivePrefix))
			Expect(updateConfig).To(BeFalse())
		})

		it("does not generate a prefix in interactive mode when auto-create-new-thread is disabled", func() {
			cfg.AutoCreateNewThread = doNotCreateNewThread
			result, updateConfig := utils.GenerateThreadName(cfg, useInteractiveMode, doNotCreateNewThread)
			Expect(result).To(Equal(threadName))
			Expect(updateConfig).To(BeFalse())
		})

		it("should return the configured thread when in command mode when auto-create-new-thread is enabled", func() {
			cfg.AutoCreateNewThread = createNewThread
			result, updateConfig := utils.GenerateThreadName(cfg, doNotUseInteractiveMode, doNotCreateNewThread)
			Expect(result).To(Equal(threadName))
			Expect(updateConfig).To(BeFalse())
		})
	})

	when("ParseMCPParams()", func() {
		const (
			key   = "key"
			value = "value"
			pair  = key + "=" + value
		)

		it("throws and error when the params are not valid JSON or a valid pair", func() {
			_, err := utils.ParseMCPParams("invalid-params")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(utils.InvalidParams))
		})
		it("parses the input as expected when a valid pair is provided", func() {
			result, err := utils.ParseMCPParams(pair)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[key]).To(Equal(value))
		})
		it("parses the input as expected when a valid json is provided", func() {
			jsonInput := `{"key": "value"}`

			result, err := utils.ParseMCPParams(jsonInput)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result["key"]).To(Equal("value"))
		})
		it("does not throw an error when no input is provided", func() {
			result, err := utils.ParseMCPParams()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
		it("throws an error when the 2nd pair is malformed", func() {
			_, err := utils.ParseMCPParams([]string{pair, "invalid-pair"}...)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(utils.InvalidParams))
		})
		it("produces the expected output when multiple pairs are provided", func() {
			result, err := utils.ParseMCPParams(pair, fmt.Sprintf("%s2=%s2", key, value))
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
			Expect(result[key]).To(Equal(value))
			Expect(result[key+"2"]).To(Equal(value + "2"))
		})
		it("parses key=value pairs where the value is a JSON array or boolean", func() {
			result, err := utils.ParseMCPParams(
				`locations=["Brooklyn","Queens"]`,
				`forecasts=true`,
				`language="en"`,
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(HaveLen(3))

			Expect(result["locations"]).To(Equal([]interface{}{"Brooklyn", "Queens"}))
			Expect(result["forecasts"]).To(Equal(true))
			Expect(result["language"]).To(Equal("en")) // NOTE: quoted value gets parsed as string
		})
	})

	when("ParseMCPHeaders()", func() {
		it("does not throw an error when no input is provided", func() {
			result, err := utils.ParseMCPHeaders(nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		it("parses a single header as expected", func() {
			result, err := utils.ParseMCPHeaders([]string{"Authorization: Bearer token"})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result["Authorization"]).To(Equal("Bearer token"))
		})

		it("trims whitespace around the key and value", func() {
			result, err := utils.ParseMCPHeaders([]string{"  Accept  :  application/json, text/event-stream  "})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result["Accept"]).To(Equal("application/json, text/event-stream"))
		})

		it("supports values that contain ':' by splitting only on the first ':'", func() {
			result, err := utils.ParseMCPHeaders([]string{"X-Test: a:b:c"})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result["X-Test"]).To(Equal("a:b:c"))
		})

		it("overwrites earlier values when the same header key is provided multiple times", func() {
			result, err := utils.ParseMCPHeaders([]string{
				"Accept: application/json",
				"Accept: application/json, text/event-stream",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result["Accept"]).To(Equal("application/json, text/event-stream"))
		})

		it("throws an error when the header is missing ':'", func() {
			_, err := utils.ParseMCPHeaders([]string{"invalid-header"})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("invalid --mcp-header %q (expected 'Key: Value')", "invalid-header")))
		})

		it("throws an error when the key is empty after trimming", func() {
			_, err := utils.ParseMCPHeaders([]string{": value"})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("invalid --mcp-header %q (empty key)", ": value")))
		})
	})
}
