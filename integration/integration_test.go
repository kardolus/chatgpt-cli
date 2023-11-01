package integration_test

import (
	"fmt"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/configmanager"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/kardolus/chatgpt-cli/utils"
	"github.com/onsi/gomega/gexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

const (
	gitCommit   = "some-git-commit"
	gitVersion  = "some-git-version"
	servicePort = ":8080"
	serviceURL  = "http://0.0.0.0" + servicePort
)

var once sync.Once

func TestIntegration(t *testing.T) {
	defer gexec.CleanupBuildArtifacts()
	spec.Run(t, "Integration Tests", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("Read, Write and Delete History", func() {
		const threadName = "default-thread"

		var (
			tmpDir   string
			fileIO   *history.FileIO
			messages []types.Message
			err      error
		)

		it.Before(func() {
			tmpDir, err = os.MkdirTemp("", "chatgpt-cli-test")
			Expect(err).NotTo(HaveOccurred())

			fileIO, _ = history.New()
			fileIO = fileIO.WithDirectory(tmpDir)
			fileIO.SetThread(threadName)

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

			_, err = os.Stat(threadName + ".json")
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

	when("Performing the Lifecycle", func() {
		const (
			exitSuccess = 0
			exitFailure = 1
		)

		var (
			homeDir      string
			err          error
			apiKeyEnvVar string
		)

		it.Before(func() {
			once.Do(func() {
				SetDefaultEventuallyTimeout(10 * time.Second)

				log.Println("Building binary...")
				Expect(buildBinary()).To(Succeed())
				log.Println("Binary built successfully!")

				log.Println("Starting mock server...")
				Expect(runMockServer()).To(Succeed())
				log.Println("Mock server started!")

				Eventually(func() (string, error) {
					return curl(fmt.Sprintf("%s/ping", serviceURL))
				}).Should(ContainSubstring("pong"))
			})

			homeDir, err = os.MkdirTemp("", "mockHome")
			Expect(err).NotTo(HaveOccurred())

			apiKeyEnvVar = configmanager.New(config.New()).WithEnvironment().APIKeyEnvVarName()

			Expect(os.Setenv("HOME", homeDir)).To(Succeed())
			Expect(os.Setenv(apiKeyEnvVar, expectedToken)).To(Succeed())
		})

		it.After(func() {
			gexec.Kill()
			Expect(os.RemoveAll(homeDir))
		})

		it("throws an error when the API key is missing", func() {
			Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

			command := exec.Command(binaryPath, "some prompt")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(apiKeyEnvVar))
		})

		it("should not require an API key for the --version flag", func() {
			Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

			command := exec.Command(binaryPath, "--version")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitSuccess))
		})

		it("should require a hidden folder for the --clear-history flag", func() {
			Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

			command := exec.Command(binaryPath, "--clear-history")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(".chatgpt-cli: no such file or directory"))
		})

		it("should require an argument for the --set-model flag", func() {
			command := exec.Command(binaryPath, "--set-model")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("flag needs an argument: --set-model"))
		})

		it("should require an argument for the --set-max-tokens flag", func() {
			command := exec.Command(binaryPath, "--set-max-tokens")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("flag needs an argument: --set-max-tokens"))
		})

		it("should require the chatgpt-cli folder but not an API key for the --set-model flag", func() {
			Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

			command := exec.Command(binaryPath, "--set-model", "123")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(".chatgpt-cli/config.yaml: no such file or directory"))
			Expect(output).NotTo(ContainSubstring(apiKeyEnvVar))
		})

		it("should require the chatgpt-cli folder but not an API key for the --set-max-tokens flag", func() {
			Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

			command := exec.Command(binaryPath, "--set-max-tokens", "789")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(".chatgpt-cli/config.yaml: no such file or directory"))
			Expect(output).NotTo(ContainSubstring(apiKeyEnvVar))
		})

		it("should return the expected result for the --version flag", func() {
			command := exec.Command(binaryPath, "--version")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitSuccess))

			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring(fmt.Sprintf("commit %s", gitCommit)))
			Expect(output).To(ContainSubstring(fmt.Sprintf("version %s", gitVersion)))
		})

		it("should return the expected result for the --list-models flag", func() {
			command := exec.Command(binaryPath, "--list-models")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitSuccess))

			output := string(session.Out.Contents())

			// see models.json
			Expect(output).To(ContainSubstring("* gpt-3.5-turbo (current)"))
			Expect(output).To(ContainSubstring("- gpt-3.5-turbo-0301"))
		})

		it("should return the expected result for the --query flag", func() {
			command := exec.Command(binaryPath, "--query", "some-query")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitSuccess))

			output := string(session.Out.Contents())

			// see completions.json
			Expect(output).To(ContainSubstring(`I don't have personal opinions about bars, but here are some popular bars in Red Hook, Brooklyn:`))
		})

		it("should assemble http errors as expected", func() {
			Expect(os.Setenv(apiKeyEnvVar, "wrong-token")).To(Succeed())

			command := exec.Command(binaryPath, "--query", "some-query")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Out.Contents())

			// see error.json
			Expect(output).To(Equal("http status 401: Incorrect API key provided\n"))
		})

		when("there is a hidden chatgpt-cli folder in the home dir", func() {
			var filePath string

			it.Before(func() {
				filePath = path.Join(os.Getenv("HOME"), ".chatgpt-cli")
				Expect(os.Mkdir(filePath, 0777)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(filePath)).To(Succeed())
			})

			it("migrates the legacy history as expected", func() {
				// Legacy history file should not exist
				legacyFile := path.Join(filePath, "history")
				Expect(legacyFile).NotTo(BeAnExistingFile())

				// History should not exist yet
				historyFile := path.Join(filePath, "history", "default.json")
				Expect(historyFile).NotTo(BeAnExistingFile())

				bytes, err := utils.FileToBytes("history.json")
				Expect(err).NotTo(HaveOccurred())

				Expect(os.WriteFile(legacyFile, bytes, 0644)).To(Succeed())
				Expect(legacyFile).To(BeARegularFile())

				// Perform a query
				command := exec.Command(binaryPath, "--query", "some-query")
				session, err := gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				// The CLI response should be as expected
				Eventually(session).Should(gexec.Exit(exitSuccess))

				output := string(session.Out.Contents())

				response := `I don't have personal opinions about bars, but here are some popular bars in Red Hook, Brooklyn:`
				Expect(output).To(ContainSubstring(response))

				// The history file should have the expected content
				Expect(path.Dir(historyFile)).To(BeADirectory())
				content, err := os.ReadFile(historyFile)

				Expect(err).NotTo(HaveOccurred())
				Expect(content).NotTo(BeEmpty())
				Expect(string(content)).To(ContainSubstring(response))

				// The legacy file should now be a directory
				Expect(legacyFile).To(BeADirectory())
				Expect(legacyFile).NotTo(BeARegularFile())

				// The content was moved to the new file
				Expect(string(content)).To(ContainSubstring("Of course! Which city are you referring to?"))
			})

			it("should not require an API key for the --clear-history flag", func() {
				Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

				command := exec.Command(binaryPath, "--clear-history")
				session, err := gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))
			})

			it("keeps track of history", func() {
				// History should not exist yet
				historyFile := path.Join(filePath, "history", "default.json")
				Expect(historyFile).NotTo(BeAnExistingFile())

				// Perform a query
				command := exec.Command(binaryPath, "--query", "some-query")
				session, err := gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				// The CLI response should be as expected
				Eventually(session).Should(gexec.Exit(exitSuccess))

				output := string(session.Out.Contents())

				response := `I don't have personal opinions about bars, but here are some popular bars in Red Hook, Brooklyn:`
				Expect(output).To(ContainSubstring(response))

				// The history file should have the expected content
				Expect(path.Dir(historyFile)).To(BeADirectory())
				content, err := os.ReadFile(historyFile)

				Expect(err).NotTo(HaveOccurred())
				Expect(content).NotTo(BeEmpty())
				Expect(string(content)).To(ContainSubstring(response))

				// Clear the history using the CLI
				command = exec.Command(binaryPath, "--clear-history")
				session, err = gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))

				// History should no longer exist
				Expect(historyFile).NotTo(BeAnExistingFile())

				// environment takes precedence
				omitHistoryEnvKey := strings.Replace(apiKeyEnvVar, "API_KEY", "OMIT_HISTORY", 1)
				envValue := "true"

				Expect(os.Setenv(omitHistoryEnvKey, envValue)).To(Succeed())

				// Perform a query
				command = exec.Command(binaryPath, "--query", "some-query")
				session, err = gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				// The CLI response should be as expected
				Eventually(session).Should(gexec.Exit(exitSuccess))

				output = string(session.Out.Contents())

				response = `I don't have personal opinions about bars, but here are some popular bars in Red Hook, Brooklyn:`
				Expect(output).To(ContainSubstring(response))

				// The history file should NOT exist
				Expect(historyFile).NotTo(BeAnExistingFile())
			})

			it("has a configurable default model", func() {
				oldModel := "gpt-3.5-turbo"
				newModel := "gpt-3.5-turbo-0301"

				// config.yaml should not exist yet
				configFile := path.Join(filePath, "config.yaml")
				Expect(configFile).NotTo(BeAnExistingFile())

				// --list-models returns the default model
				command := exec.Command(binaryPath, "--list-models")
				session, err := gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))

				output := string(session.Out.Contents())

				// see models.json
				Expect(output).To(ContainSubstring(fmt.Sprintf("* %s (current)", oldModel)))
				Expect(output).To(ContainSubstring(fmt.Sprintf("- %s", newModel)))

				// --config displays the default model as well
				command = exec.Command(binaryPath, "--config")
				session, err = gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))

				output = string(session.Out.Contents())

				Expect(output).To(ContainSubstring(oldModel))

				// Set the model
				command = exec.Command(binaryPath, "--set-model", newModel)
				session, err = gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				// The CLI response should be as expected
				Eventually(session).Should(gexec.Exit(exitSuccess))

				// config.yaml should have been created
				Expect(configFile).To(BeAnExistingFile())

				contentBytes, err := os.ReadFile(configFile)
				Expect(err).ShouldNot(HaveOccurred())

				// config.yaml should have the expected content
				content := string(contentBytes)
				Expect(content).NotTo(ContainSubstring(expectedToken))
				Expect(content).To(ContainSubstring(newModel))

				// --list-models shows the new model as default
				command = exec.Command(binaryPath, "--list-models")
				session, err = gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))

				output = string(session.Out.Contents())

				Expect(output).To(ContainSubstring(fmt.Sprintf("- %s", oldModel)))
				Expect(output).To(ContainSubstring(fmt.Sprintf("* %s (current)", newModel)))

				// --config displays the new model as well
				command = exec.Command(binaryPath, "--config")
				session, err = gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))

				output = string(session.Out.Contents())
				Expect(output).To(ContainSubstring(newModel))

				// environment takes precedence
				modelEnvKey := strings.Replace(apiKeyEnvVar, "API_KEY", "MODEL", 1)
				envModel := "new-model"

				Expect(os.Setenv(modelEnvKey, envModel)).To(Succeed())

				command = exec.Command(binaryPath, "--config")
				session, err = gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))

				output = string(session.Out.Contents())
				Expect(output).To(ContainSubstring(envModel))

				Expect(os.Unsetenv(modelEnvKey)).To(Succeed())
			})

			it("has a configurable default max-tokens", func() {
				defaults := config.New().ReadDefaults()

				// config.yaml should not exist yet
				configFile := path.Join(filePath, "config.yaml")
				Expect(configFile).NotTo(BeAnExistingFile())

				// --config displays the default max-tokens as well
				command := exec.Command(binaryPath, "--config")
				session, err := gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))

				output := string(session.Out.Contents())

				Expect(output).To(ContainSubstring(strconv.Itoa(defaults.MaxTokens)))

				// Set the max-tokens
				newMaxTokens := "81724"
				command = exec.Command(binaryPath, "--set-max-tokens", newMaxTokens)
				session, err = gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				// The CLI response should be as expected
				Eventually(session).Should(gexec.Exit(exitSuccess))

				// config.yaml should have been created
				Expect(configFile).To(BeAnExistingFile())

				contentBytes, err := os.ReadFile(configFile)
				Expect(err).ShouldNot(HaveOccurred())

				// config.yaml should have the expected content
				content := string(contentBytes)
				Expect(content).NotTo(ContainSubstring(expectedToken))
				Expect(content).To(ContainSubstring(newMaxTokens))

				// --config displays the new max-tokens as well
				command = exec.Command(binaryPath, "--config")
				session, err = gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))

				output = string(session.Out.Contents())
				Expect(output).To(ContainSubstring(newMaxTokens))

				// environment takes precedence
				modelEnvKey := strings.Replace(apiKeyEnvVar, "API_KEY", "MAX_TOKENS", 1)

				Expect(os.Setenv(modelEnvKey, newMaxTokens)).To(Succeed())

				command = exec.Command(binaryPath, "--config")
				session, err = gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))

				output = string(session.Out.Contents())
				Expect(output).To(ContainSubstring(newMaxTokens))

				Expect(os.Unsetenv(modelEnvKey)).To(Succeed())
			})
		})
	})
}
