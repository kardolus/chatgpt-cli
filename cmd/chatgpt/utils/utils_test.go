package utils_test

import (
	"fmt"
	"github.com/kardolus/chatgpt-cli/cmd/chatgpt/utils"
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
}
