package test

import (
	"embed"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

var (
	//go:embed resources/Dockerfile
	DockerfileBytes []byte

	// Include file prefixed with '_' explicitly
	//go:embed resources/testchart/*
	//go:embed resources/testchart/templates/_helpers.tpl
	TestChart embed.FS
)

func CopyDir(srcFS embed.FS, srcPath string, destPath string) error {
	dirEntries, err := srcFS.ReadDir(srcPath)
	if err != nil {
		return fmt.Errorf("Error reading %s: %v", srcPath, err)
	}
	for _, dirEntry := range dirEntries {
		srcItemPath := path.Join(srcPath, dirEntry.Name())
		destItemPath := path.Join(destPath, dirEntry.Name())
		srcItemInfo, err := dirEntry.Info()
		if err != nil {
			return fmt.Errorf("Error getting info for %s: %v", dirEntry.Name(), err)
		}
		if dirEntry.IsDir() {
			err = os.Mkdir(destItemPath, srcItemInfo.Mode()|0700)
			if err != nil {
				return fmt.Errorf("Error making dir %s: %v", destItemPath, err)
			}
			err = CopyDir(srcFS, srcItemPath, destItemPath)
			if err != nil {
				return fmt.Errorf("Error copying from %s to %s: %v", srcItemPath, destItemPath, err)
			}
		} else {
			content, err := srcFS.ReadFile(srcItemPath)
			if err != nil {
				return fmt.Errorf("Error reading file %s: %v", srcItemPath, err)
			}
			err = ioutil.WriteFile(destItemPath, content, srcItemInfo.Mode()|0600)
			if err != nil {
				return fmt.Errorf("Error writing to file %s: %v", destItemPath, err)
			}
		}
	}
	return nil
}
