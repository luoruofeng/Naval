package util

import (
	"os"
	"path/filepath"
)

func GetRootProjPath(current string) string {
	path := filepath.Join(current, "go.mod")
	_, err := os.Stat(path)
	if err == nil {
		return current
	} else {
		if os.IsNotExist(err) {
			return GetRootProjPath(filepath.Dir(current))
		} else {
			panic(err)
		}
	}
}
