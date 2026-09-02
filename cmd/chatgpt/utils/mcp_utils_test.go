package utils_test

import (
	"os"
	"testing"

	"github.com/kardolus/chatgpt-cli/cmd/chatgpt/utils"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestMCPUtils(t *testing.T) {
	spec.Run(t, "Testing MCP utilities", testMCPUtils, spec.Report(report.Terminal{}))
}

func testMCPUtils(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("ResolveMCPEndpoint", func() {
		it("resolves 'you' to You.com MCP endpoint", func() {
			result := utils.ResolveMCPEndpoint("you")
			Expect(result).To(Equal("https://api.you.com/mcp"))
		})

		it("resolves 'youcom' to You.com MCP endpoint", func() {
			result := utils.ResolveMCPEndpoint("youcom")
			Expect(result).To(Equal("https://api.you.com/mcp"))
		})

		it("resolves 'you.com' to You.com MCP endpoint", func() {
			result := utils.ResolveMCPEndpoint("you.com")
			Expect(result).To(Equal("https://api.you.com/mcp"))
		})

		it("handles case-insensitive input", func() {
			Expect(utils.ResolveMCPEndpoint("YOU")).To(Equal("https://api.you.com/mcp"))
			Expect(utils.ResolveMCPEndpoint("YouCom")).To(Equal("https://api.you.com/mcp"))
		})

		it("handles whitespace", func() {
			result := utils.ResolveMCPEndpoint("  you  ")
			Expect(result).To(Equal("https://api.you.com/mcp"))
		})

		it("returns custom endpoints unchanged", func() {
			custom := "http://localhost:3000/mcp"
			result := utils.ResolveMCPEndpoint(custom)
			Expect(result).To(Equal(custom))
		})

		it("returns empty string unchanged", func() {
			result := utils.ResolveMCPEndpoint("")
			Expect(result).To(Equal(""))
		})
	})

	when("BuildYouComHeaders", func() {
		it("returns empty headers when no API key is set", func() {
			// Ensure no API key is set
			os.Unsetenv("YDC_API_KEY")
			headers := utils.BuildYouComHeaders()
			Expect(headers).To(BeEmpty())
		})

		it("adds Authorization header when API key is set", func() {
			testKey := "test-api-key-12345"
			os.Setenv("YDC_API_KEY", testKey)
			defer os.Unsetenv("YDC_API_KEY")

			headers := utils.BuildYouComHeaders()
			Expect(headers).To(HaveKey("Authorization"))
			Expect(headers["Authorization"]).To(Equal("Bearer " + testKey))
		})
	})

	when("IsYouComEndpoint", func() {
		it("identifies You.com endpoint correctly", func() {
			Expect(utils.IsYouComEndpoint("https://api.you.com/mcp")).To(BeTrue())
		})

		it("identifies resolved shortcuts as You.com endpoint", func() {
			Expect(utils.IsYouComEndpoint("you")).To(BeTrue())
			Expect(utils.IsYouComEndpoint("youcom")).To(BeTrue())
			Expect(utils.IsYouComEndpoint("you.com")).To(BeTrue())
		})

		it("identifies other endpoints as not You.com", func() {
			Expect(utils.IsYouComEndpoint("http://localhost:3000")).To(BeFalse())
			Expect(utils.IsYouComEndpoint("https://example.com/mcp")).To(BeFalse())
			Expect(utils.IsYouComEndpoint("")).To(BeFalse())
		})
	})
}