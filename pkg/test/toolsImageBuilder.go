package test

import (
	"archive/tar"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const ToolsImageName = "aks-periscope-test-tools"

// Include file prefixed with '_' explicitly
//go:embed resources/*
//go:embed resources/testchart/templates/_helpers.tpl
var resources embed.FS

type ToolsImageBuilder struct {
	client *client.Client
}

func NewToolsImageBuilder(client *client.Client) *ToolsImageBuilder {
	return &ToolsImageBuilder{
		client: client,
	}
}

func (builder *ToolsImageBuilder) Build() error {
	ctx := context.Background()

	archiveContent, err := createArchive()
	if err != nil {
		return fmt.Errorf("Error creating resources archive: %v", err)
	}

	dockerFileTarReader := bytes.NewReader(archiveContent)

	buildOptions := types.ImageBuildOptions{
		Context:    dockerFileTarReader,
		Dockerfile: "Dockerfile",
		Remove:     true,
		Tags:       []string{ToolsImageName},
	}

	imageBuildResponse, err := builder.client.ImageBuild(ctx, dockerFileTarReader, buildOptions)
	if err != nil {
		return fmt.Errorf("Error building docker image: %v", err)
	}

	defer imageBuildResponse.Body.Close()

	// Read the STDOUT from the build process
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		return fmt.Errorf("Error copying build output to stdout: %v", err)
	}

	return nil
}

func createArchive() ([]byte, error) {
	buffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(buffer)
	defer tarWriter.Close()

	err := addToArchive(tarWriter, resources, "resources", "")
	if err != nil {
		return nil, fmt.Errorf("Error creating archive for resources: %v", err)
	}

	return buffer.Bytes(), nil
}

func addToArchive(tarWriter *tar.Writer, srcFS embed.FS, srcDirPath, destDirPath string) error {
	dirEntries, err := srcFS.ReadDir(srcDirPath)
	if err != nil {
		return fmt.Errorf("Error reading directory %s: %v", srcDirPath, err)
	}
	for _, dirEntry := range dirEntries {
		srcItemPath := path.Join(srcDirPath, dirEntry.Name())
		destItemPath := path.Join(destDirPath, dirEntry.Name())

		srcItemInfo, err := dirEntry.Info()
		if err != nil {
			return fmt.Errorf("Error getting info for %s: %v", srcItemPath, err)
		}

		tarHeader, err := tar.FileInfoHeader(srcItemInfo, "")
		if err != nil {
			return fmt.Errorf("Error creating tar header for %s: %v", destItemPath, err)
		}

		tarHeader.Name = destItemPath
		if dirEntry.IsDir() {
			tarHeader.Name += "/"
		}

		err = tarWriter.WriteHeader(tarHeader)
		if err != nil {
			return fmt.Errorf("Error writing tar header for %s: %v", destItemPath, err)
		}

		if dirEntry.IsDir() {
			if err = addToArchive(tarWriter, srcFS, srcItemPath, destItemPath); err != nil {
				return err
			}
		} else {
			content, err := srcFS.ReadFile(srcItemPath)
			if err != nil {
				return fmt.Errorf("Error reading file %s: %v", srcItemPath, err)
			}

			_, err = tarWriter.Write(content)
			if err != nil {
				return fmt.Errorf("Error writing file file content to archive %s: %v", destItemPath, err)
			}
		}
	}
	return nil
}
