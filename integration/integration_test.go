package integration_test

import (
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io/ioutil"
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

	when("Read, Write and Delete", func() {
		var (
			tmpDir   string
			tmpFile  *os.File
			fileIO   *history.FileIO
			messages []types.Message
			err      error
		)

		it.Before(func() {
			tmpDir, err = ioutil.TempDir("", "chatgpt-cli-test")
			Expect(err).NotTo(HaveOccurred())

			tmpFile, err = ioutil.TempFile(tmpDir, "history.json")
			Expect(err).NotTo(HaveOccurred())

			tmpFile.Close()

			fileIO = history.New().WithHistory(tmpFile.Name())

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
			os.RemoveAll(tmpDir)
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
}
