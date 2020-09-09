package utils

import (
	"bytes"
	"io/ioutil"
	"os"
)

// ReplacePathInFile modify the path in config.yaml, replace `GOPATH` by goPath
func ReplacePathInFile(config string) []byte {
	input, err := ioutil.ReadFile(config)
	if err != nil {
		Fatalf("ReadFile err:", err)
	}

	goPath := os.Getenv("GOPATH")

	return bytes.ReplaceAll(input, []byte("GOPATH"), []byte(goPath))
}
