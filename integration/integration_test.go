package integration_test

import (
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	spec.Run(t, "Integration Tests", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("Read, Write and Delete History", func() {
		var (
			tmpDir   string
			tmpFile  *os.File
			fileIO   *history.FileIO
			messages []types.Message
			err      error
		)

		it.Before(func() {
			tmpDir, err = os.MkdirTemp("", "chatgpt-cli-test")
			Expect(err).NotTo(HaveOccurred())

			tmpFile, err = os.CreateTemp(tmpDir, "history.json")
			Expect(err).NotTo(HaveOccurred())

			Expect(tmpFile.Close()).To(Succeed())

			fileIO = history.New().WithFilePath(tmpFile.Name())

			messages = []types.Message{
				{
					Role:    "user",
					Content: "Test message 1",
				},
				{
					Role:    "assistant",
					Content: "Test message 2",
				},
			}
		})

		it.After(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		it("writes the messages to the file", func() {
			err = fileIO.Write(messages)
			Expect(err).NotTo(HaveOccurred())
		})

		it("reads the messages from the file", func() {
			err = fileIO.Write(messages) // need to write before reading
			Expect(err).NotTo(HaveOccurred())

			readMessages, err := fileIO.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(readMessages).To(Equal(messages))
		})

		it("deletes the file", func() {
			err = fileIO.Delete()
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(tmpFile.Name())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	when("Read, Write Config", func() {
		var (
			tmpDir     string
			tmpFile    *os.File
			configIO   *config.FileIO
			testConfig types.Config
			err        error
		)

		it.Before(func() {
			tmpDir, err = os.MkdirTemp("", "chatgpt-cli-test")
			Expect(err).NotTo(HaveOccurred())

			tmpFile, err = os.CreateTemp(tmpDir, "config.yaml")
			Expect(err).NotTo(HaveOccurred())

			Expect(tmpFile.Close()).To(Succeed())

			configIO = config.New().WithFilePath(tmpFile.Name())

			testConfig = types.Config{
				Model: "test-model",
			}
		})

		it.After(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		it("writes the config to the file", func() {
			err = configIO.Write(testConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		it("reads the config from the file", func() {
			err = configIO.Write(testConfig) // need to write before reading
			Expect(err).NotTo(HaveOccurred())

			readConfig, err := configIO.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(readConfig).To(Equal(testConfig))
		})

		// Since we don't have a Delete method in the config, we will test if we can overwrite the configuration.
		it("overwrites the existing config", func() {
			newConfig := types.Config{
				Model: "new-model",
			}
			err = configIO.Write(newConfig)
			Expect(err).NotTo(HaveOccurred())

			readConfig, err := configIO.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(readConfig).To(Equal(newConfig))
		})
	})

}
