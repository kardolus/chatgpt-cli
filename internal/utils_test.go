package internal_test

import (
	"github.com/kardolus/chatgpt-cli/internal"
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
		Expect(os.Unsetenv(internal.ConfigHomeEnv)).To(Succeed())
		Expect(os.Unsetenv(internal.DataHomeEnv)).To(Succeed())
		Expect(os.Unsetenv(internal.CacheHomeEnv)).To(Succeed())
	})

	when("GetConfigHome()", func() {
		it("Uses the default value if OPENAI_CONFIG_HOME is not set", func() {
			configHome, err := internal.GetConfigHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(configHome).To(ContainSubstring(".chatgpt-cli")) // Assuming default location is ~/.chatgpt-cli
		})

		it("Overwrites the default when OPENAI_CONFIG_HOME is set", func() {
			customConfigHome := "/custom/config/path"
			Expect(os.Setenv("OPENAI_CONFIG_HOME", customConfigHome)).To(Succeed())

			configHome, err := internal.GetConfigHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(configHome).To(Equal(customConfigHome))
		})
	})

	when("GetDataHome()", func() {
		it("Uses the default value if OPENAI_DATA_HOME is not set", func() {
			dataHome, err := internal.GetDataHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(dataHome).To(ContainSubstring(".chatgpt-cli/history")) // Assuming default location is ~/.local/share/chatgpt-cli
		})

		it("Overwrites the default when OPENAI_DATA_HOME is set", func() {
			customDataHome := "/custom/data/path"
			Expect(os.Setenv("OPENAI_DATA_HOME", customDataHome)).To(Succeed())

			dataHome, err := internal.GetDataHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(dataHome).To(Equal(customDataHome))
		})
	})

	when("GetCacheHome()", func() {
		it("Uses the default value if OPENAI_CACHE_HOME is not set", func() {
			cacheHome, err := internal.GetCacheHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(cacheHome).To(ContainSubstring(".chatgpt-cli/cache")) // Assuming default location is ~/.local/share/chatgpt-cli
		})

		it("Overwrites the default when OPENAI_CACHE_HOME is set", func() {
			cacheDataHome := "/custom/cache/path"
			Expect(os.Setenv("OPENAI_CACHE_HOME", cacheDataHome)).To(Succeed())

			cacheHome, err := internal.GetCacheHome()

			Expect(err).NotTo(HaveOccurred())
			Expect(cacheHome).To(Equal(cacheDataHome))
		})
	})

	when("GenerateUniqueSlug()", func() {
		it("Has the expected length", func() {
			prefix := "123"
			result := internal.GenerateUniqueSlug(prefix)
			Expect(result).To(HaveLen(len(prefix) + internal.SlugPostfixLength))
		})
	})
}
