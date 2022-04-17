package test

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const ToolsImageName = "aks-periscope-test-tools"

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

	// Create a buffer
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Make a TAR header for the dockerfile
	tarHeader := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(DockerfileBytes)),
	}

	// Write the header and content
	err := tw.WriteHeader(tarHeader)
	if err != nil {
		return err
	}

	_, err = tw.Write(DockerfileBytes)
	if err != nil {
		return err
	}

	dockerFileTarReader := bytes.NewReader(buf.Bytes())

	// Define the build options to use for the file
	// https://godoc.org/github.com/docker/docker/api/types#ImageBuildOptions
	buildOptions := types.ImageBuildOptions{
		Context:    dockerFileTarReader,
		Dockerfile: "Dockerfile",
		Remove:     true,
		Tags:       []string{ToolsImageName},
	}

	// Build the actual image
	imageBuildResponse, err := builder.client.ImageBuild(
		ctx,
		dockerFileTarReader,
		buildOptions,
	)

	if err != nil {
		return err
	}

	// Read the STDOUT from the build process
	defer imageBuildResponse.Body.Close()
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		return err
	}

	return nil
}
