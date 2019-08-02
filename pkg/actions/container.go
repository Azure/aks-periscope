package actions

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/Azure/aks-diagnostic-tool/pkg/utils"
)

// PollContainerLogs poll container logs in namespace
func PollContainerLogs(podNameSpace string) ([]string, error) {
	containerNames := make([]string, 0)
	containerLogs := make([]string, 0)

	rootPath := filepath.Join("/aks-diagnostic", utils.GetHostName())
	err := os.MkdirAll(rootPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	for _, container := range containers {
		parts := strings.Split(container.Names[0], "_")
		if parts[1] != "POD" && parts[3] == podNameSpace {
			containerNames = append(containerNames, strings.TrimPrefix(container.Names[0], "/"))
		}
	}

	for _, containerName := range containerNames {
		out, err := cli.ContainerLogs(ctx, containerName, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		containerLog := filepath.Join(rootPath, containerName)
		file, err := os.Create(containerLog)
		defer file.Close()
		io.Copy(file, out)

		containerLogs = append(containerLogs, containerLog)
	}

	return containerLogs, nil
}
