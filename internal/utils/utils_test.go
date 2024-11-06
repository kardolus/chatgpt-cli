package utils_test

import (
	"github.com/kardolus/chatgpt-cli/internal/utils"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"os"
	"testing"
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

	when("GetConfigHome()", func() {
		it("Uses the default value if OPENAI_CONFIG_HOME is not set", func() {
			configHome, err := utils.GetConfigHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(configHome).To(ContainSubstring(".chatgpt-cli")) // Assuming default location is ~/.chatgpt-cli
		})

		it("Overwrites the default when OPENAI_CONFIG_HOME is set", func() {
			customConfigHome := "/custom/config/path"
			Expect(os.Setenv("OPENAI_CONFIG_HOME", customConfigHome)).To(Succeed())

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
			Expect(os.Setenv("OPENAI_DATA_HOME", customDataHome)).To(Succeed())

			dataHome, err := utils.GetDataHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(dataHome).To(Equal(customDataHome))
		})
	})
}
