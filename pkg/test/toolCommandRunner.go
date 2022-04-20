package test

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type ToolsCommandRunner struct {
	client *client.Client
}

func NewToolsCommandRunner(client *client.Client) *ToolsCommandRunner {
	return &ToolsCommandRunner{
		client: client,
	}
}

func (creator *ToolsCommandRunner) Run(command string, volumeBinds ...string) (string, error) {

	// https://godoc.org/github.com/docker/docker/api/types/container#Config
	config := &container.Config{
		Image: ToolsImageName,
		Cmd:   []string{"sh", "-c", command},
	}

	// https://godoc.org/github.com/docker/docker/api/types/container#HostConfig
	hostConfig := &container.HostConfig{
		Binds:       append(volumeBinds, "/var/run/docker.sock:/var/run/docker.sock"),
		NetworkMode: "host",
	}
	cont, err := creator.client.ContainerCreate(
		context.Background(),
		config,
		hostConfig,
		nil,
		nil,
		ToolsImageName,
	)

	if err != nil {
		return "", fmt.Errorf("Failed to create container\nCommand: %s\nError: %v", command, err)
	}

	// Remove container after running command, whether successful or not.
	// There is an auto-remove option when creating the container, but we avoid this because
	// it introduces a race condition while we wait for the container.
	defer removeContainer(creator.client, cont.ID)

	err = creator.client.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", fmt.Errorf("Failed to create container\nCommand: %s\nError: %v", command, err)
	}

	waitResultChan, errChan := creator.client.ContainerWait(context.Background(), cont.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errChan:
		return "", fmt.Errorf("Failed waiting for command %s: %v", command, err)
	case result := <-waitResultChan:
		if result.StatusCode != 0 {
			stdout, stderr, err := getContainerLogs(creator.client, cont.ID)
			if err != nil {
				return "", fmt.Errorf("Command failed with status %d, but unable to read container logs.\nCommand: %s\nError: %v", result.StatusCode, command, err)
			}
			return "", fmt.Errorf("Command failed with status %d\nCommand: %s\nStdout: %s\nStderr: %s", result.StatusCode, command, stdout, stderr)
		}
	}

	stdout, _, err := getContainerLogs(creator.client, cont.ID)
	if err != nil {
		return "", fmt.Errorf("Unable to read container logs.\nCommand: %s\nError: %v", command, err)
	}

	return stdout, nil
}

func removeContainer(client *client.Client, containerId string) {
	err := client.ContainerRemove(context.Background(), containerId, types.ContainerRemoveOptions{})
	if err != nil {
		log.Printf("Error removing container ID %s", containerId)
	}
}

func getContainerLogs(client *client.Client, containerId string) (string, string, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}

	body, err := client.ContainerLogs(context.Background(), containerId, options)
	if err != nil {
		return "", "", fmt.Errorf("Error getting logs: %v", err)
	}

	defer body.Close()

	var stdOutBuff, stdErrBuff bytes.Buffer
	_, err = stdcopy.StdCopy(&stdOutBuff, &stdErrBuff, body)
	if err != nil {
		return "", "", fmt.Errorf("Error reading logs: %v", err)
	}

	return stdOutBuff.String(), stdErrBuff.String(), nil
}
