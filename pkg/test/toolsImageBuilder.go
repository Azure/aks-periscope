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
//go:embed resources/Dockerfile
//go:embed resources/tools-resources/*
//go:embed resources/tools-resources/testchart/templates/_helpers.tpl
var resources embed.FS

// ToolsImageBuilder provides a method for building the Docker image that contains all the tools
// involved in initializing a Kind cluster for tests.
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
		return fmt.Errorf("error creating resources archive: %w", err)
	}

	dockerFileTarReader := bytes.NewReader(archiveContent)

	osmVersionVar := osmVersion // need a variable here, because we can't get a pointer to a const string
	buildOptions := types.ImageBuildOptions{
		Context:    dockerFileTarReader,
		Dockerfile: "Dockerfile",
		Remove:     true,
		Tags:       []string{ToolsImageName},
		BuildArgs: map[string]*string{
			"OSM_VERSION": &osmVersionVar,
		},
	}

	imageBuildResponse, err := builder.client.ImageBuild(ctx, dockerFileTarReader, buildOptions)
	if err != nil {
		return fmt.Errorf("error building docker image: %w", err)
	}

	defer imageBuildResponse.Body.Close()

	// Read the STDOUT from the build process
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		return fmt.Errorf("error copying build output to stdout: %w", err)
	}

	return nil
}

func createArchive() ([]byte, error) {
	buffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(buffer)
	defer tarWriter.Close()

	err := addToArchive(tarWriter, resources, "resources", "")
	if err != nil {
		return nil, fmt.Errorf("error creating archive for resources: %w", err)
	}

	return buffer.Bytes(), nil
}

func addToArchive(tarWriter *tar.Writer, srcFS embed.FS, srcDirPath, destDirPath string) error {
	dirEntries, err := srcFS.ReadDir(srcDirPath)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", srcDirPath, err)
	}
	for _, dirEntry := range dirEntries {
		srcItemPath := path.Join(srcDirPath, dirEntry.Name())
		destItemPath := path.Join(destDirPath, dirEntry.Name())

		srcItemInfo, err := dirEntry.Info()
		if err != nil {
			return fmt.Errorf("error getting info for %s: %w", srcItemPath, err)
		}

		tarHeader, err := tar.FileInfoHeader(srcItemInfo, "")
		if err != nil {
			return fmt.Errorf("error creating tar header for %s: %w", destItemPath, err)
		}

		tarHeader.Name = destItemPath
		if dirEntry.IsDir() {
			tarHeader.Name += "/"
		}

		err = tarWriter.WriteHeader(tarHeader)
		if err != nil {
			return fmt.Errorf("error writing tar header for %s: %w", destItemPath, err)
		}

		if dirEntry.IsDir() {
			if err = addToArchive(tarWriter, srcFS, srcItemPath, destItemPath); err != nil {
				return err
			}
		} else {
			content, err := srcFS.ReadFile(srcItemPath)
			if err != nil {
				return fmt.Errorf("error reading file %s: %w", srcItemPath, err)
			}

			_, err = tarWriter.Write(content)
			if err != nil {
				return fmt.Errorf("error writing file file content to archive %s: %w", destItemPath, err)
			}
		}
	}
	return nil
}
