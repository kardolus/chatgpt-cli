package utils

import (
	. "github.com/onsi/gomega"
	"io/ioutil"
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
		urlPath, err = filepath.Abs(path.Join(thisFile, "../../../../../..", "resources", "testdata", fileName))
	} else {
		urlPath, err = filepath.Abs(path.Join(thisFile, "../..", "resources", "testdata", fileName))
	}

	if err != nil {
		return nil, err
	}

	Expect(urlPath).To(BeAnExistingFile())

	return ioutil.ReadFile(urlPath)
}
