package integration_test

import (
	"encoding/json"
	"fmt"
	"github.com/kardolus/chatgpt-cli/agent"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/cache"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/internal"
	"github.com/kardolus/chatgpt-cli/test"
	"github.com/onsi/gomega/gexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

var (
	once sync.Once
)

func TestIntegration(t *testing.T) {
	defer gexec.CleanupBuildArtifacts()
	spec.Run(t, "Integration Tests", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
		Expect(os.Unsetenv(internal.ConfigHomeEnv)).To(Succeed())
		Expect(os.Unsetenv(internal.DataHomeEnv)).To(Succeed())
	})

	when("Read and Write History", func() {
		const threadName = "default-thread"

		var (
			tmpDir   string
			fileIO   *history.FileIO
			messages []api.Message
			err      error
		)

		it.Before(func() {
			tmpDir, err = os.MkdirTemp("", "chatgpt-cli-test")
			Expect(err).NotTo(HaveOccurred())

			fileIO, _ = history.New()
			fileIO = fileIO.WithDirectory(tmpDir)
			fileIO.SetThread(threadName)

			messages = []api.Message{
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
			var historyEntries []history.History
			for _, message := range messages {
				historyEntries = append(historyEntries, history.History{
					Message: message,
				})
			}

			err = fileIO.Write(historyEntries)
			Expect(err).NotTo(HaveOccurred())
		})

		it("reads the messages from the file", func() {
			var historyEntries []history.History
			for _, message := range messages {
				historyEntries = append(historyEntries, history.History{
					Message: message,
				})
			}

			err = fileIO.Write(historyEntries) // need to write before reading
			Expect(err).NotTo(HaveOccurred())

			readEntries, err := fileIO.Read()
			Expect(err).NotTo(HaveOccurred())
			Expect(readEntries).To(Equal(historyEntries))
		})
	})

	when("Read, Write and Delete Cache", func() {
		var (
			tmpDir   string
			storeDir string
			err      error
		)

		const endpoint = "http://127.0.0.1:8000/mcp"

		it.Before(func() {
			tmpDir, err = os.MkdirTemp("", "chatgpt-cli-cache-test")
			Expect(err).NotTo(HaveOccurred())

			// Simulate what will likely become ~/.chatgpt-cli/cache/mcp/sessions
			storeDir = filepath.Join(tmpDir, "cache", "mcp", "sessions")
		})

		it.After(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		it("writes, reads, and deletes a session id", func() {
			fs := cache.NewFileStore(storeDir)
			c := cache.New(fs)

			Expect(c.SetSessionID(endpoint, "sid-1")).To(Succeed())

			got, err := c.GetSessionID(endpoint)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal("sid-1"))

			Expect(c.DeleteSessionID(endpoint)).To(Succeed())

			// After delete, Get should error (os.ErrNotExist bubbling up)
			_, err = c.GetSessionID(endpoint)
			Expect(err).To(HaveOccurred())
		})

		it("persists across cache instances (simulates separate CLI invocations)", func() {
			fs1 := cache.NewFileStore(storeDir)
			c1 := cache.New(fs1)

			Expect(c1.SetSessionID(endpoint, "sid-abc")).To(Succeed())

			// New instances, same underlying directory
			fs2 := cache.NewFileStore(storeDir)
			c2 := cache.New(fs2)

			got, err := c2.GetSessionID(endpoint)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal("sid-abc"))
		})

		it("overwrites an existing session id (rotation)", func() {
			fs := cache.NewFileStore(storeDir)
			c := cache.New(fs)

			Expect(c.SetSessionID(endpoint, "sid-old")).To(Succeed())
			Expect(c.SetSessionID(endpoint, "sid-new")).To(Succeed())

			got, err := c.GetSessionID(endpoint)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal("sid-new"))
		})
	})

	when("Read, Write, List, Delete Config", func() {
		var (
			tmpDir     string
			tmpFile    *os.File
			historyDir string
			configIO   *config.FileIO
			testConfig config.Config
			err        error
		)

		it.Before(func() {
			tmpDir, err = os.MkdirTemp("", "chatgpt-cli-test")
			Expect(err).NotTo(HaveOccurred())

			historyDir, err = os.MkdirTemp(tmpDir, "history")
			Expect(err).NotTo(HaveOccurred())

			tmpFile, err = os.CreateTemp(tmpDir, "config.yaml")
			Expect(err).NotTo(HaveOccurred())

			Expect(tmpFile.Close()).To(Succeed())

			configIO = config.NewStore().WithConfigPath(tmpFile.Name()).WithHistoryPath(historyDir)

			testConfig = config.Config{
				Model: "test-model",
			}
		})

		it.After(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		when("performing a migration", func() {
			defaults := config.NewStore().ReadDefaults()

			it("it doesn't apply a migration when max_tokens is 0", func() {
				testConfig.MaxTokens = 0

				err = configIO.Write(testConfig) // need to write before reading
				Expect(err).NotTo(HaveOccurred())

				readConfig, err := configIO.Read()
				Expect(err).NotTo(HaveOccurred())
				Expect(readConfig).To(Equal(testConfig))
			})
			it("it migrates small values of max_tokens as expected", func() {
				testConfig.MaxTokens = defaults.ContextWindow - 1

				err = configIO.Write(testConfig) // need to write before reading
				Expect(err).NotTo(HaveOccurred())

				readConfig, err := configIO.Read()
				Expect(err).NotTo(HaveOccurred())

				expectedConfig := testConfig
				expectedConfig.MaxTokens = defaults.MaxTokens
				expectedConfig.ContextWindow = defaults.ContextWindow

				Expect(readConfig).To(Equal(expectedConfig))
			})
			it("it migrates large values of max_tokens as expected", func() {
				testConfig.MaxTokens = defaults.ContextWindow + 1

				err = configIO.Write(testConfig) // need to write before reading
				Expect(err).NotTo(HaveOccurred())

				readConfig, err := configIO.Read()
				Expect(err).NotTo(HaveOccurred())

				expectedConfig := testConfig
				expectedConfig.MaxTokens = defaults.MaxTokens
				expectedConfig.ContextWindow = testConfig.MaxTokens

				Expect(readConfig).To(Equal(expectedConfig))
			})
		})

		it("lists all the threads", func() {
			files := []string{"thread1.json", "thread2.json", "thread3.json"}

			for _, file := range files {
				file, err := os.Create(filepath.Join(historyDir, file))
				Expect(err).NotTo(HaveOccurred())

				Expect(file.Close()).To(Succeed())
			}

			result, err := configIO.List()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(3))
			Expect(result[2]).To(Equal("thread3.json"))
		})

		it("deletes the thread", func() {
			files := []string{"thread1.json", "thread2.json", "thread3.json"}

			for _, file := range files {
				file, err := os.Create(filepath.Join(historyDir, file))
				Expect(err).NotTo(HaveOccurred())

				Expect(file.Close()).To(Succeed())
			}

			err = configIO.Delete("thread2")
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(filepath.Join(historyDir, "thread2.json"))
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Stat(filepath.Join(historyDir, "thread3.json"))
			Expect(os.IsNotExist(err)).To(BeFalse())
		})
	})

	when("Performing the Lifecycle", func() {
		const (
			exitSuccess = 0
			exitFailure = 1
		)

		var (
			homeDir      string
			filePath     string
			configFile   string
			err          error
			apiKeyEnvVar string
		)

		runCommand := func(args ...string) string {
			command := exec.Command(binaryPath, args...)
			session, err := gexec.Start(command, io.Discard, io.Discard)

			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			<-session.Exited

			if tmp := string(session.Err.Contents()); tmp != "" {
				fmt.Printf("error output: %s", string(session.Err.Contents()))
			}

			ExpectWithOffset(1, session).Should(gexec.Exit(0))
			return string(session.Out.Contents())
		}

		runCommandWithStdin := func(stdin io.Reader, args ...string) string {
			command := exec.Command(binaryPath, args...)
			command.Stdin = stdin
			session, err := gexec.Start(command, io.Discard, io.Discard)

			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			<-session.Exited

			if tmp := string(session.Err.Contents()); tmp != "" {
				fmt.Printf("error output: %s", tmp)
			}

			ExpectWithOffset(1, session).Should(gexec.Exit(0))
			return string(session.Out.Contents())
		}

		checkConfigFileContent := func(expectedContent string) {
			content, err := os.ReadFile(configFile)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			ExpectWithOffset(1, string(content)).To(ContainSubstring(expectedContent))
		}

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

			apiKeyEnvVar = config.NewManager(config.NewStore()).WithEnvironment().APIKeyEnvVarName()

			Expect(os.Setenv("HOME", homeDir)).To(Succeed())
			Expect(os.Setenv(apiKeyEnvVar, expectedToken)).To(Succeed())
		})

		it.After(func() {
			gexec.Kill()
			Expect(os.RemoveAll(homeDir))
		})

		when("resolving the API key", func() {
			var secretFile string

			it.Before(func() {
				secretFile = filepath.Join(homeDir, ".chatgpt-cli", "secret.key")
				Expect(os.MkdirAll(filepath.Dir(secretFile), 0700)).To(Succeed())
				Expect(os.WriteFile(secretFile, []byte(expectedToken+"\n"), 0600)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(filepath.Dir(secretFile))).To(Succeed())
				Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())
				Expect(os.Unsetenv("OPENAI_API_KEY_FILE")).To(Succeed())
			})

			it("prefers the API key from environment variable over the file", func() {
				Expect(os.Setenv(apiKeyEnvVar, "env-api-key")).To(Succeed())
				Expect(os.Setenv("OPENAI_API_KEY_FILE", secretFile)).To(Succeed())

				cmd := exec.Command(binaryPath, "--config")
				session, err := gexec.Start(cmd, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(exitSuccess))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("env-api-key"))
			})

			it("uses the file if env var is not set", func() {
				Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())
				Expect(os.Setenv("OPENAI_API_KEY_FILE", secretFile)).To(Succeed())

				cmd := exec.Command(binaryPath, "--list-models")
				session, err := gexec.Start(cmd, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(exitSuccess))
			})

			it("errors if neither env var nor file is set", func() {
				Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())
				Expect(os.Unsetenv("OPENAI_API_KEY_FILE")).To(Succeed())

				cmd := exec.Command(binaryPath, "--list-models")
				session, err := gexec.Start(cmd, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(exitFailure))
				errOutput := string(session.Err.Contents())
				Expect(errOutput).To(ContainSubstring("API key is required"))
			})
		})

		it("should not require an API key for the --version flag", func() {
			Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

			command := exec.Command(binaryPath, "--version")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitSuccess))
		})

		it("should require a hidden folder for the --list-threads flag", func() {
			command := exec.Command(binaryPath, "--list-threads")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring(".chatgpt-cli/history: no such file or directory"))
		})

		it("should return an error when --new-thread is used with --set-thread", func() {
			command := exec.Command(binaryPath, "--new-thread", "--set-thread", "some-thread")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring("the --new-thread flag cannot be used with the --set-thread or --thread flags"))
		})

		it("should return an error when --new-thread is used with --thread", func() {
			command := exec.Command(binaryPath, "--new-thread", "--thread", "some-thread")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring("the --new-thread flag cannot be used with the --set-thread or --thread flags"))
		})

		it("should require an argument for the --set-model flag", func() {
			command := exec.Command(binaryPath, "--set-model")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring("flag needs an argument: --set-model"))
		})

		it("should require an argument for the --set-thread flag", func() {
			command := exec.Command(binaryPath, "--set-thread")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring("flag needs an argument: --set-thread"))
		})

		it("should require an argument for the --set-max-tokens flag", func() {
			command := exec.Command(binaryPath, "--set-max-tokens")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring("flag needs an argument: --set-max-tokens"))
		})

		it("should require an argument for the --set-context-window flag", func() {
			command := exec.Command(binaryPath, "--set-context-window")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring("flag needs an argument: --set-context-window"))
		})

		it("should warn when config.yaml does not exist and OPENAI_CONFIG_HOME is set", func() {
			configHomeDir := "does-not-exist"
			Expect(os.Setenv(internal.ConfigHomeEnv, configHomeDir)).To(Succeed())

			configFilePath := path.Join(configHomeDir, "config.yaml")
			Expect(configFilePath).NotTo(BeAnExistingFile())

			command := exec.Command(binaryPath, "llm query")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitSuccess))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: config.yaml doesn't exist in %s, create it", configHomeDir)))

			// Unset the variable to prevent pollution
			Expect(os.Unsetenv(internal.ConfigHomeEnv)).To(Succeed())
		})

		it("should NOT warn when config.yaml does not exist and OPENAI_CONFIG_HOME is NOT set", func() {
			configHomeDir := "does-not-exist"
			Expect(os.Unsetenv(internal.ConfigHomeEnv)).To(Succeed())

			configFilePath := path.Join(configHomeDir, "config.yaml")
			Expect(configFilePath).NotTo(BeAnExistingFile())

			command := exec.Command(binaryPath, "llm query")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitSuccess))

			output := string(session.Out.Contents())
			Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: config.yaml doesn't exist in %s, create it", configHomeDir)))
		})

		it("should require the chatgpt-cli folder but not an API key for the --set-model flag", func() {
			Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

			command := exec.Command(binaryPath, "--set-model", "123")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring("config directory does not exist:"))
			Expect(output).NotTo(ContainSubstring(apiKeyEnvVar))
		})

		it("should require the chatgpt-cli folder but not an API key for the --set-thread flag", func() {
			Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

			command := exec.Command(binaryPath, "--set-thread", "thread-name")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring("config directory does not exist:"))
			Expect(output).NotTo(ContainSubstring(apiKeyEnvVar))
		})

		it("should require the chatgpt-cli folder but not an API key for the --set-max-tokens flag", func() {
			Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

			command := exec.Command(binaryPath, "--set-max-tokens", "789")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring("config directory does not exist:"))
			Expect(output).NotTo(ContainSubstring(apiKeyEnvVar))
		})

		it("should require the chatgpt-cli folder but not an API key for the --set-context-window flag", func() {
			Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

			command := exec.Command(binaryPath, "--set-context-window", "789")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())
			Expect(output).To(ContainSubstring("config directory does not exist:"))
			Expect(output).NotTo(ContainSubstring(apiKeyEnvVar))
		})

		it("should return the expected result for the --version flag", func() {
			output := runCommand("--version")

			Expect(output).To(ContainSubstring(fmt.Sprintf("commit %s", gitCommit)))
			Expect(output).To(ContainSubstring(fmt.Sprintf("version %s", gitVersion)))
		})

		it("should return the expected result for the --list-models flag", func() {
			output := runCommand("--list-models")

			Expect(output).To(ContainSubstring("* gpt-4o (current)"))
			Expect(output).To(ContainSubstring("- gpt-3.5-turbo"))
			Expect(output).To(ContainSubstring("- gpt-3.5-turbo-0301"))
		})

		it("should return the expected result for the --query flag", func() {
			Expect(os.Setenv("OPENAI_TRACK_TOKEN_USAGE", "false")).To(Succeed())

			output := runCommand("--query", "some-query")

			expectedResponse := `I don't have personal opinions about bars, but here are some popular bars in Red Hook, Brooklyn:`
			Expect(output).To(ContainSubstring(expectedResponse))
			Expect(output).NotTo(ContainSubstring("Token Usage:"))
		})

		it("should display token usage after a query when configured to do so", func() {
			Expect(os.Setenv("OPENAI_TRACK_TOKEN_USAGE", "true")).To(Succeed())

			output := runCommand("--query", "tell me a 5 line joke")
			Expect(output).To(ContainSubstring("Token Usage:"))
		})

		it("prints debug information with the --debug flag", func() {
			output := runCommand("--query", "tell me a joke", "--debug")

			Expect(output).To(ContainSubstring("Generated cURL command"))
			Expect(output).To(ContainSubstring("/v1/chat/completions"))
			Expect(output).To(ContainSubstring("--header \"Authorization: Bearer ${OPENAI_API_KEY}\""))
			Expect(output).To(ContainSubstring("--header 'Content-Type: application/json'"))
			Expect(output).To(ContainSubstring("--header 'User-Agent: chatgpt-cli'"))
			Expect(output).To(ContainSubstring("\"model\":\"gpt-4o\""))
			Expect(output).To(ContainSubstring("\"messages\":"))
			Expect(output).To(ContainSubstring("Response"))

			Expect(os.Unsetenv("OPENAI_DEBUG")).To(Succeed())
		})

		it("should assemble http errors as expected", func() {
			Expect(os.Setenv(apiKeyEnvVar, "wrong-token")).To(Succeed())

			command := exec.Command(binaryPath, "--query", "some-query")
			session, err := gexec.Start(command, io.Discard, io.Discard)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gexec.Exit(exitFailure))

			output := string(session.Err.Contents())

			// see error.json
			Expect(output).To(Equal("http status 401: Incorrect API key provided\n"))
		})

		when("loading configuration via --target", func() {
			var (
				configDir    string
				mainConfig   string
				targetConfig string
			)

			it.Before(func() {
				RegisterTestingT(t)

				var err error
				configDir, err = os.MkdirTemp("", "chatgpt-cli-test")
				Expect(err).NotTo(HaveOccurred())

				Expect(os.Setenv("OPENAI_CONFIG_HOME", configDir)).To(Succeed())

				mainConfig = filepath.Join(configDir, "config.yaml")
				targetConfig = filepath.Join(configDir, "config.testtarget.yaml")

				Expect(os.WriteFile(mainConfig, []byte("model: gpt-4o\n"), 0644)).To(Succeed())
				Expect(os.WriteFile(targetConfig, []byte("model: gpt-3.5-turbo-0301\n"), 0644)).To(Succeed())
			})

			it("should load config.testtarget.yaml when using --target", func() {
				cmd := exec.Command(binaryPath, "--target", "testtarget", "--config")

				session, err := gexec.Start(cmd, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("gpt-3.5-turbo-0301"))
			})

			it("should fall back to config.yaml when --target is not used", func() {
				cmd := exec.Command(binaryPath, "--config")

				session, err := gexec.Start(cmd, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(0))
				output := string(session.Out.Contents())
				Expect(output).To(ContainSubstring("gpt-4o"))
			})
		})

		when("there is a hidden chatgpt-cli folder in the home dir", func() {
			it.Before(func() {
				filePath = path.Join(os.Getenv("HOME"), ".chatgpt-cli")
				Expect(os.MkdirAll(filePath, 0777)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(filePath)).To(Succeed())
			})

			it("should not require an API key for the --list-threads flag", func() {
				historyPath := path.Join(filePath, "history")
				Expect(os.MkdirAll(historyPath, 0777)).To(Succeed())

				Expect(os.Unsetenv(apiKeyEnvVar)).To(Succeed())

				command := exec.Command(binaryPath, "--list-threads")
				session, err := gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))
			})

			it("migrates the legacy history as expected", func() {
				// Legacy history file should not exist
				legacyFile := path.Join(filePath, "history")
				Expect(legacyFile).NotTo(BeAnExistingFile())

				// History should not exist yet
				historyFile := path.Join(filePath, "history", "default.json")
				Expect(historyFile).NotTo(BeAnExistingFile())

				bytes, err := test.FileToBytes("history.json")
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
				historyDir := path.Join(filePath, "history")
				historyFile := path.Join(historyDir, "default.json")
				Expect(historyFile).NotTo(BeAnExistingFile())

				// Perform a query and check response
				response := `I don't have personal opinions about bars, but here are some popular bars in Red Hook, Brooklyn:`
				output := runCommand("--query", "some-query")
				Expect(output).To(ContainSubstring(response))

				// Check if history file was created with expected content
				Expect(historyDir).To(BeADirectory())
				checkHistoryContent := func(expectedContent string) {
					content, err := os.ReadFile(historyFile)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(content)).To(ContainSubstring(expectedContent))
				}
				checkHistoryContent(response)

				// Clear the history using the CLI
				runCommand("--clear-history")
				Expect(historyFile).NotTo(BeAnExistingFile())

				// Test omitting history through environment variable
				omitHistoryEnvKey := strings.Replace(apiKeyEnvVar, "API_KEY", "OMIT_HISTORY", 1)
				envValue := "true"
				Expect(os.Setenv(omitHistoryEnvKey, envValue)).To(Succeed())

				// Perform another query with history omitted
				runCommand("--query", "some-query")
				// The history file should NOT be recreated
				Expect(historyFile).NotTo(BeAnExistingFile())

				// Cleanup: Unset the environment variable
				Expect(os.Unsetenv(omitHistoryEnvKey)).To(Succeed())
			})

			it("should not add binary data to the history", func() {
				historyDir := path.Join(filePath, "history")
				historyFile := path.Join(historyDir, "default.json")
				Expect(historyFile).NotTo(BeAnExistingFile())

				response := `I don't have personal opinions about bars, but here are some popular bars in Red Hook, Brooklyn:`

				// Create a pipe to simulate binary input
				r, w := io.Pipe()
				defer r.Close()

				// Run the command with piped binary input
				binaryData := []byte{0x00, 0xFF, 0x42, 0x10}
				go func() {
					defer w.Close()
					_, err := w.Write(binaryData)
					Expect(err).NotTo(HaveOccurred())
				}()

				// Run the command with stdin redirected
				output := runCommandWithStdin(r, "--query", "some-query")
				Expect(output).To(ContainSubstring(response))

				Expect(historyDir).To(BeADirectory())
				checkHistoryContent := func(expectedContent string) {
					content, err := os.ReadFile(historyFile)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(content)).To(ContainSubstring(expectedContent))
				}
				checkHistoryContent(response)
			})

			it("should return the expected result for the --list-threads flag", func() {
				historyDir := path.Join(filePath, "history")
				Expect(os.Mkdir(historyDir, 0755)).To(Succeed())

				files := []string{"thread1.json", "thread2.json", "thread3.json", "default.json"}

				Expect(os.MkdirAll(historyDir, 7555)).To(Succeed())

				for _, file := range files {
					file, err := os.Create(filepath.Join(historyDir, file))
					Expect(err).NotTo(HaveOccurred())

					Expect(file.Close()).To(Succeed())
				}

				output := runCommand("--list-threads")

				Expect(output).To(ContainSubstring("* default (current)"))
				Expect(output).To(ContainSubstring("- thread1"))
				Expect(output).To(ContainSubstring("- thread2"))
				Expect(output).To(ContainSubstring("- thread3"))
			})

			it("should delete the expected thread using the --delete-threads flag", func() {
				historyDir := path.Join(filePath, "history")
				Expect(os.Mkdir(historyDir, 0755)).To(Succeed())

				files := []string{"thread1.json", "thread2.json", "thread3.json", "default.json"}

				Expect(os.MkdirAll(historyDir, 7555)).To(Succeed())

				for _, file := range files {
					file, err := os.Create(filepath.Join(historyDir, file))
					Expect(err).NotTo(HaveOccurred())

					Expect(file.Close()).To(Succeed())
				}

				runCommand("--delete-thread", "thread2")

				output := runCommand("--list-threads")

				Expect(output).To(ContainSubstring("* default (current)"))
				Expect(output).To(ContainSubstring("- thread1"))
				Expect(output).NotTo(ContainSubstring("- thread2"))
				Expect(output).To(ContainSubstring("- thread3"))
			})

			it("should delete the expected threads using the --delete-threads flag with a wildcard", func() {
				historyDir := filepath.Join(filePath, "history")
				Expect(os.Mkdir(historyDir, 0755)).To(Succeed())

				files := []string{
					"start1.json", "start2.json", "start3.json",
					"1end.json", "2end.json", "3end.json",
					"1middle1.json", "2middle2.json", "3middle3.json",
					"other1.json", "other2.json",
				}

				createTestFiles := func(dir string, filenames []string) {
					for _, filename := range filenames {
						file, err := os.Create(filepath.Join(dir, filename))
						Expect(err).NotTo(HaveOccurred())
						Expect(file.Close()).To(Succeed())
					}
				}

				createTestFiles(historyDir, files)

				output := runCommand("--list-threads")
				expectedThreads := []string{
					"start1", "start2", "start3",
					"1end", "2end", "3end",
					"1middle1", "2middle2", "3middle3",
					"other1", "other2",
				}
				for _, thread := range expectedThreads {
					Expect(output).To(ContainSubstring("- " + thread))
				}

				tests := []struct {
					pattern        string
					remainingAfter []string
				}{
					{"start*", []string{"1end", "2end", "3end", "1middle1", "2middle2", "3middle3", "other1", "other2"}},
					{"*end", []string{"1middle1", "2middle2", "3middle3", "other1", "other2"}},
					{"*middle*", []string{"other1", "other2"}},
					{"*", []string{}}, // Should delete all remaining threads
				}

				for _, tt := range tests {
					runCommand("--delete-thread", tt.pattern)
					output = runCommand("--list-threads")

					for _, thread := range tt.remainingAfter {
						Expect(output).To(ContainSubstring("- " + thread))
					}
				}
			})

			it("should throw an error when a non-existent thread is deleted using the --delete-threads flag", func() {
				command := exec.Command(binaryPath, "--delete-thread", "does-not-exist")
				session, err := gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitFailure))
			})

			it("should not throw an error --clear-history is called without there being a history", func() {
				command := exec.Command(binaryPath, "--clear-history")
				session, err := gexec.Start(command, io.Discard, io.Discard)
				Expect(err).NotTo(HaveOccurred())

				Eventually(session).Should(gexec.Exit(exitSuccess))
			})

			when("configurable flags are set", func() {
				it.Before(func() {
					configFile = path.Join(filePath, "config.yaml")
					Expect(configFile).NotTo(BeAnExistingFile())
				})

				it("has a configurable default model", func() {
					oldModel := "gpt-4o"
					newModel := "gpt-3.5-turbo-0301"

					// Verify initial model
					output := runCommand("--list-models")
					Expect(output).To(ContainSubstring("* " + oldModel + " (current)"))
					Expect(output).To(ContainSubstring("- " + newModel))

					// Update model
					runCommand("--set-model", newModel)

					// Check configFile is created and contains the new model
					Expect(configFile).To(BeAnExistingFile())
					checkConfigFileContent(newModel)

					// Verify updated model through --list-models
					output = runCommand("--list-models")

					Expect(output).To(ContainSubstring("* " + newModel + " (current)"))
				})

				it("has a configurable default context-window", func() {
					defaults := config.NewStore().ReadDefaults()

					// Initial check for default context-window
					output := runCommand("--config")
					Expect(output).To(ContainSubstring(strconv.Itoa(defaults.ContextWindow)))

					// Update and verify context-window
					newContextWindow := "100000"
					runCommand("--set-context-window", newContextWindow)
					Expect(configFile).To(BeAnExistingFile())
					checkConfigFileContent(newContextWindow)

					// Verify update through --config
					output = runCommand("--config")
					Expect(output).To(ContainSubstring(newContextWindow))

					// Environment variable takes precedence
					envContext := "123"
					modelEnvKey := strings.Replace(apiKeyEnvVar, "API_KEY", "CONTEXT_WINDOW", 1)
					Expect(os.Setenv(modelEnvKey, envContext)).To(Succeed())

					// Verify environment variable override
					output = runCommand("--config")
					Expect(output).To(ContainSubstring(envContext))
					Expect(os.Unsetenv(modelEnvKey)).To(Succeed())
				})

				it("has a configurable default max-tokens", func() {
					defaults := config.NewStore().ReadDefaults()

					// Initial check for default max-tokens
					output := runCommand("--config")
					Expect(output).To(ContainSubstring(strconv.Itoa(defaults.MaxTokens)))

					// Update and verify max-tokens
					newMaxTokens := "81724"
					runCommand("--set-max-tokens", newMaxTokens)
					Expect(configFile).To(BeAnExistingFile())
					checkConfigFileContent(newMaxTokens)

					// Verify update through --config
					output = runCommand("--config")
					Expect(output).To(ContainSubstring(newMaxTokens))

					// Environment variable takes precedence
					modelEnvKey := strings.Replace(apiKeyEnvVar, "API_KEY", "MAX_TOKENS", 1)
					Expect(os.Setenv(modelEnvKey, newMaxTokens)).To(Succeed())

					// Verify environment variable override
					output = runCommand("--config")
					Expect(output).To(ContainSubstring(newMaxTokens))
					Expect(os.Unsetenv(modelEnvKey)).To(Succeed())
				})

				it("has a configurable default thread", func() {
					defaults := config.NewStore().ReadDefaults()

					// Initial check for default thread
					output := runCommand("--config")
					Expect(output).To(ContainSubstring(defaults.Thread))

					// Update and verify thread
					newThread := "new-thread"
					runCommand("--set-thread", newThread)
					Expect(configFile).To(BeAnExistingFile())
					checkConfigFileContent(newThread)

					// Verify update through --config
					output = runCommand("--config")
					Expect(output).To(ContainSubstring(newThread))

					// Environment variable takes precedence
					threadEnvKey := strings.Replace(apiKeyEnvVar, "API_KEY", "THREAD", 1)
					Expect(os.Setenv(threadEnvKey, newThread)).To(Succeed())

					// Verify environment variable override
					output = runCommand("--config")
					Expect(output).To(ContainSubstring(newThread))
					Expect(os.Unsetenv(threadEnvKey)).To(Succeed())
				})
			})
		})

		when("configuration precedence", func() {
			var (
				defaultModel = "gpt-4o"
				newModel     = "gpt-3.5-turbo-0301"
				envModel     = "gpt-3.5-env-model"
				envVar       string
			)

			it.Before(func() {
				envVar = strings.Replace(apiKeyEnvVar, "API_KEY", "MODEL", 1)
				filePath = path.Join(os.Getenv("HOME"), ".chatgpt-cli")
				Expect(os.MkdirAll(filePath, 0777)).To(Succeed())

				configFile = path.Join(filePath, "config.yaml")
				Expect(configFile).NotTo(BeAnExistingFile())
			})

			it("uses environment variable over config file", func() {
				// Step 1: Set a model in the config file.
				runCommand("--set-model", newModel)
				checkConfigFileContent(newModel)

				// Step 2: Verify the model from config is used.
				output := runCommand("--list-models")
				Expect(output).To(ContainSubstring("* " + newModel + " (current)"))

				// Step 3: Set environment variable and verify it takes precedence.
				Expect(os.Setenv(envVar, envModel)).To(Succeed())
				output = runCommand("--list-models")
				Expect(output).To(ContainSubstring("* " + envModel + " (current)"))

				// Step 4: Unset environment variable and verify it falls back to config file.
				Expect(os.Unsetenv(envVar)).To(Succeed())
				output = runCommand("--list-models")
				Expect(output).To(ContainSubstring("* " + newModel + " (current)"))
			})

			it("uses command-line flag over environment variable", func() {
				// Step 1: Set environment variable.
				Expect(os.Setenv(envVar, envModel)).To(Succeed())

				// Step 2: Verify environment variable does not override flag.
				output := runCommand("--model", newModel, "--list-models")
				Expect(output).To(ContainSubstring("* " + newModel + " (current)"))
			})

			it("falls back to default when config and env are absent", func() {
				// Step 1: Ensure no config file and no environment variable.
				Expect(os.Unsetenv(envVar)).To(Succeed())

				// Step 2: Verify it falls back to the default model.
				output := runCommand("--list-models")
				Expect(output).To(ContainSubstring("* " + defaultModel + " (current)"))
			})
		})

		when("show-history flag is used", func() {
			var tmpDir string
			var err error
			var historyFile string

			it.Before(func() {
				RegisterTestingT(t)
				tmpDir, err = os.MkdirTemp("", "chatgpt-cli-test")
				Expect(err).NotTo(HaveOccurred())
				historyFile = filepath.Join(tmpDir, "default.json")

				messages := []api.Message{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi, how can I help you?"},
					{Role: "user", Content: "Tell me about the weather"},
					{Role: "assistant", Content: "It's sunny today."},
				}
				data, err := json.Marshal(messages)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.WriteFile(historyFile, data, 0644)).To(Succeed())

				// This is legacy: we need a config dir in order to have a history dir
				filePath = path.Join(os.Getenv("HOME"), ".chatgpt-cli")
				Expect(os.MkdirAll(filePath, 0777)).To(Succeed())

				Expect(os.Setenv("OPENAI_DATA_HOME", tmpDir)).To(Succeed())
			})

			it("prints the history for the default thread", func() {
				output := runCommand("--show-history")

				// Check that the output contains the history as expected
				Expect(output).To(ContainSubstring("**USER** 👤:\nHello"))
				Expect(output).To(ContainSubstring("**ASSISTANT** 🤖:\nHi, how can I help you?"))
				Expect(output).To(ContainSubstring("**USER** 👤:\nTell me about the weather"))
				Expect(output).To(ContainSubstring("**ASSISTANT** 🤖:\nIt's sunny today."))
			})

			it("prints the history for a specific thread when specified", func() {
				specificThread := "specific-thread"
				specificHistoryFile := filepath.Join(tmpDir, specificThread+".json")

				// Create a specific thread with custom history
				messages := []api.Message{
					{Role: "user", Content: "What's the capital of Belgium?"},
					{Role: "assistant", Content: "The capital of Belgium is Brussels."},
				}
				data, err := json.Marshal(messages)
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(specificHistoryFile, data, 0644)).To(Succeed())

				// Run the --show-history flag with the specific thread
				output := runCommand("--show-history", specificThread)

				// Check that the output contains the history as expected
				Expect(output).To(ContainSubstring("**USER** 👤:\nWhat's the capital of Belgium?"))
				Expect(output).To(ContainSubstring("**ASSISTANT** 🤖:\nThe capital of Belgium is Brussels."))
			})

			it("concatenates user messages correctly", func() {
				// Create history where two user messages are concatenated
				messages := []api.Message{
					{Role: "user", Content: "Part one"},
					{Role: "user", Content: " and part two"},
					{Role: "assistant", Content: "This is a response."},
				}
				data, err := json.Marshal(messages)
				Expect(err).NotTo(HaveOccurred())
				Expect(os.WriteFile(historyFile, data, 0644)).To(Succeed())

				output := runCommand("--show-history")

				// Check that the concatenated user messages are displayed correctly
				Expect(output).To(ContainSubstring("**USER** 👤:\nPart one and part two"))
				Expect(output).To(ContainSubstring("**ASSISTANT** 🤖:\nThis is a response."))
			})
		})
	})

	when("Agent Files ops", func() {
		var (
			tmpDir string
			err    error
			ops    agent.FSIOFileOps
		)

		it.Before(func() {
			tmpDir, err = os.MkdirTemp("", "chatgpt-cli-files-it")
			Expect(err).NotTo(HaveOccurred())

			ops = agent.NewFSIOFileOps(osReader{}, osWriter{})
		})

		it.After(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		it("WriteFile writes and overwrites full content", func() {
			p := filepath.Join(tmpDir, "a.txt")

			Expect(ops.WriteFile(p, []byte("hello\n"))).To(Succeed())
			b, err := os.ReadFile(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(Equal("hello\n"))

			Expect(ops.WriteFile(p, []byte("goodbye\n"))).To(Succeed())
			b, err = os.ReadFile(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(Equal("goodbye\n"))
		})

		it("PatchFile applies unified diff and persists changes", func() {
			p := filepath.Join(tmpDir, "b.txt")
			Expect(os.WriteFile(p, []byte("a\nb\nc\n"), 0o644)).To(Succeed())

			diff := []byte(
				"@@ -1,3 +1,3 @@\n" +
					" a\n" +
					"-b\n" +
					"+B\n" +
					" c\n",
			)

			res, err := ops.PatchFile(p, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Hunks).To(Equal(1))

			got, err := os.ReadFile(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(got)).To(Equal("a\nB\nc\n"))
		})

		it("PatchFile is a no-op when patch produces no changes", func() {
			p := filepath.Join(tmpDir, "c.txt")
			Expect(os.WriteFile(p, []byte("a\nb\n"), 0o644)).To(Succeed())

			// Patch that effectively keeps the file identical.
			diff := []byte(
				"@@ -1,2 +1,2 @@\n" +
					" a\n" +
					" b\n",
			)

			res, err := ops.PatchFile(p, diff)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Hunks).To(Equal(1))

			got, err := os.ReadFile(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(got)).To(Equal("a\nb\n"))
		})

		it("PatchFile returns a wrapped error when patch cannot be applied", func() {
			p := filepath.Join(tmpDir, "d.txt")
			Expect(os.WriteFile(p, []byte("a\nb\nc\n"), 0o644)).To(Succeed())

			// Context mismatch: expects 'x' where file has 'b'
			diff := []byte(
				"@@ -1,3 +1,3 @@\n" +
					" a\n" +
					"-x\n" +
					"+B\n" +
					" c\n",
			)

			res, err := ops.PatchFile(p, diff)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("apply patch " + p + ":"))
			Expect(res.Hunks).To(Equal(1))
		})

		it("ReplaceBytesInFile replaces all occurrences when n<=0", func() {
			p := filepath.Join(tmpDir, "e.txt")
			Expect(os.WriteFile(p, []byte("aa bb aa bb aa\n"), 0o644)).To(Succeed())

			res, err := ops.ReplaceBytesInFile(p, []byte("aa"), []byte("XX"), 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.OccurrencesFound).To(Equal(3))
			Expect(res.Replaced).To(Equal(3))

			got, err := os.ReadFile(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(got)).To(Equal("XX bb XX bb XX\n"))
		})

		it("ReplaceBytesInFile replaces only the first n occurrences when n>0", func() {
			p := filepath.Join(tmpDir, "f.txt")
			Expect(os.WriteFile(p, []byte("x x x x\n"), 0o644)).To(Succeed())

			res, err := ops.ReplaceBytesInFile(p, []byte("x"), []byte("y"), 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.OccurrencesFound).To(Equal(4))
			Expect(res.Replaced).To(Equal(2))

			got, err := os.ReadFile(p)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(got)).To(Equal("y y x x\n"))
		})

		it("ReplaceBytesInFile errors when old pattern is empty", func() {
			p := filepath.Join(tmpDir, "g.txt")
			Expect(os.WriteFile(p, []byte("hello\n"), 0o644)).To(Succeed())

			res, err := ops.ReplaceBytesInFile(p, []byte(""), []byte("x"), -1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("old pattern must be non-empty"))
			Expect(res).To(Equal(agent.ReplaceResult{}))
		})

		it("ReplaceBytesInFile errors when pattern is not found", func() {
			p := filepath.Join(tmpDir, "h.txt")
			Expect(os.WriteFile(p, []byte("hello\n"), 0o644)).To(Succeed())

			res, err := ops.ReplaceBytesInFile(p, []byte("nope"), []byte("x"), -1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("pattern not found"))
			Expect(res.OccurrencesFound).To(Equal(0))
			Expect(res.Replaced).To(Equal(0))
		})

		it("ReplaceBytesInFile errors when replacement produces no change", func() {
			p := filepath.Join(tmpDir, "i.txt")
			Expect(os.WriteFile(p, []byte("hello hello\n"), 0o644)).To(Succeed())

			res, err := ops.ReplaceBytesInFile(p, []byte("hello"), []byte("hello"), -1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no changes applied"))
			Expect(res.OccurrencesFound).To(Equal(2))
			Expect(res.Replaced).To(Equal(0))
		})
	})
}

type osReader struct{}

func (osReader) Open(name string) (*os.File, error) {
	return os.Open(name)
}

func (osReader) ReadBufferFromFile(file *os.File) ([]byte, error) {
	return io.ReadAll(file)
}

func (r osReader) ReadFile(name string) ([]byte, error) {
	f, err := r.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return r.ReadBufferFromFile(f)
}

type osWriter struct{}

func (osWriter) Create(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.Create(path)
}

func (osWriter) Write(f *os.File, data []byte) error {
	_, err := f.Write(data)
	return err
}
