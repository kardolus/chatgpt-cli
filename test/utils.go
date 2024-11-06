package test

import (
	. "github.com/onsi/gomega"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func FileToBytes(fileName string) ([]byte, error) {
	_, thisFile, _, _ := runtime.Caller(0)

	var (
		urlPath string
		err     error
	)
	if strings.Contains(thisFile, "vendor") {
		urlPath, err = filepath.Abs(path.Join(thisFile, "../../../../../..", "test", "data", fileName))
	} else {
		urlPath, err = filepath.Abs(path.Join(thisFile, "../..", "test", "data", fileName))
	}

	if err != nil {
		return nil, err
	}

	Expect(urlPath).To(BeAnExistingFile())

	return os.ReadFile(urlPath)
}
