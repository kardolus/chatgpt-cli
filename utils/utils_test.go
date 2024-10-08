package utils_test

import (
	"fmt"
	"github.com/kardolus/chatgpt-cli/utils"
	"os"
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
		Expect(os.Unsetenv(utils.ConfigHomeEnv)).To(Succeed())
		Expect(os.Unsetenv(utils.DataHomeEnv)).To(Succeed())
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

	when("GetConfigHome()", func() {
		it("Uses the default value if OPENAI_CONFIG_HOME is not set", func() {
			configHome, err := utils.GetConfigHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(configHome).To(ContainSubstring(".chatgpt-cli")) // Assuming default location is ~/.chatgpt-cli
		})

		it("Overwrites the default when OPENAI_CONFIG_HOME is set", func() {
			customConfigHome := "/custom/config/path"
			os.Setenv("OPENAI_CONFIG_HOME", customConfigHome)

			configHome, err := utils.GetConfigHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(configHome).To(Equal(customConfigHome))
		})
	})

	when("GetDataHome()", func() {
		it("Uses the default value if OPENAI_DATA_HOME is not set", func() {
			dataHome, err := utils.GetDataHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(dataHome).To(ContainSubstring(".chatgpt-cli/history")) // Assuming default location is ~/.local/share/chatgpt-cli
		})

		it("Overwrites the default when OPENAI_DATA_HOME is set", func() {
			customDataHome := "/custom/data/path"
			os.Setenv("OPENAI_DATA_HOME", customDataHome)

			dataHome, err := utils.GetDataHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(dataHome).To(Equal(customDataHome))
		})
	})
}
