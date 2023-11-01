package integration_test

import (
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/kardolus/chatgpt-cli/utils"
	"github.com/onsi/gomega/gexec"
	"io"
	"net/http"
	"strings"
	"sync"
)

const expectedToken = "valid-api-key"

var (
	onceBuild   sync.Once
	onceServe   sync.Once
	serverReady = make(chan struct{})
	binaryPath  string
)

func buildBinary() error {
	var err error
	onceBuild.Do(func() {
		binaryPath, err = gexec.Build(
			"github.com/kardolus/chatgpt-cli/cmd/chatgpt",
			"-ldflags",
			fmt.Sprintf("-X main.GitCommit=%s -X main.GitVersion=%s -X main.ServiceURL=%s", gitCommit, gitVersion, serviceURL))
	})
	return err
}

func curl(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func runMockServer() error {
	var (
		defaults types.Config
		err      error
	)

	onceServe.Do(func() {
		go func() {
			defaults = config.New().ReadDefaults()

			http.HandleFunc("/ping", getPing)
			http.HandleFunc(defaults.CompletionsPath, postCompletions)
			http.HandleFunc(defaults.ModelsPath, getModels)
			close(serverReady)
			err = http.ListenAndServe(servicePort, nil)
		}()
	})
	<-serverReady
	return err
}

func getPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Write([]byte("pong"))
}

func getModels(w http.ResponseWriter, r *http.Request) {
	if err := validateRequest(w, r, http.MethodGet); err != nil {
		fmt.Printf("invalid request: %s\n", err.Error())
		return
	}

	if err := checkBearerToken(r, expectedToken); err != nil {
		http.Error(w, creatAuthError(), http.StatusUnauthorized)
		return
	}

	const modelFile = "models.json"
	response, err := utils.FileToBytes(modelFile)
	if err != nil {
		fmt.Printf("error reading %s: %s\n", modelFile, err.Error())
		return
	}
	w.Write(response)
}

func postCompletions(w http.ResponseWriter, r *http.Request) {
	if err := validateRequest(w, r, http.MethodPost); err != nil {
		fmt.Printf("invalid request: %s\n", err.Error())
		return
	}

	if err := checkBearerToken(r, expectedToken); err != nil {
		http.Error(w, creatAuthError(), http.StatusUnauthorized)
		return
	}

	const completionsFile = "completions.json"
	response, err := utils.FileToBytes(completionsFile)
	if err != nil {
		fmt.Printf("error reading %s: %s\n", completionsFile, err.Error())
		return
	}
	w.Write(response)
}

func checkBearerToken(r *http.Request, expectedToken string) error {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return errors.New("missing Authorization header")
	}

	splitToken := strings.Split(authHeader, "Bearer ")
	if len(splitToken) != 2 {
		return errors.New("malformed Authorization header")
	}

	requestToken := splitToken[1]
	if requestToken != expectedToken {
		return errors.New("invalid token")
	}

	return nil
}

func creatAuthError() string {
	const errorFile = "error.json"

	response, err := utils.FileToBytes(errorFile)
	if err != nil {
		fmt.Printf("error reading %s: %s\n", errorFile, err.Error())
		return ""
	}

	return string(response)
}

func validateRequest(w http.ResponseWriter, r *http.Request, allowedMethod string) error {
	if r.Method != allowedMethod {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return errors.New("method not allowed")
	}

	if !strings.Contains(r.Header.Get("Authorization"), "Bearer") {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("bad request")
	}

	return nil
}
