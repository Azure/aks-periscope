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

// ToolsCommandRunner provides a means to invoke command-line tools within a Docker container
// made for testing purposes.
type ToolsCommandRunner struct {
	client *client.Client
}

func NewToolsCommandRunner(client *client.Client) *ToolsCommandRunner {
	return &ToolsCommandRunner{
		client: client,
	}
}

// Run executest the specified command in the tools container, with the specified volume bindings.
// It returns the stdout of the executed command.
func (creator *ToolsCommandRunner) Run(command string, volumeBinds ...string) (string, error) {
	config := &container.Config{
		Image: ToolsImageName,
		Cmd:   []string{"sh", "-c", command},
	}

	// Always bind the docker socket because we're expecting to use the docker client within the container.
	// Host networking is required to connect to the cluster API server, which is exposed on a port on the host.
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
		return "", fmt.Errorf("failed to create container\nCommand: %s\nError: %w", command, err)
	}

	// Remove container after running command, whether successful or not.
	// There is an auto-remove option when creating the container, but we avoid this because
	// it introduces a race condition while we wait for the container.
	defer removeContainer(creator.client, cont.ID)

	err = creator.client.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create container\nCommand: %s\nError: %w", command, err)
	}

	waitResultChan, errChan := creator.client.ContainerWait(context.Background(), cont.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errChan:
		return "", fmt.Errorf("failed waiting for command %s: %w", command, err)
	case result := <-waitResultChan:
		if result.StatusCode != 0 {
			stdout, stderr, err := getContainerLogs(creator.client, cont.ID)
			if err != nil {
				return "", fmt.Errorf("command failed with status %d, but unable to read container logs.\nCommand: %s\nError: %w", result.StatusCode, command, err)
			}
			return "", fmt.Errorf("command failed with status %d\nCommand: %s\nStdout: %s\nStderr: %s", result.StatusCode, command, stdout, stderr)
		}
	}

	stdout, _, err := getContainerLogs(creator.client, cont.ID)
	if err != nil {
		return "", fmt.Errorf("unable to read container logs.\nCommand: %s\nError: %w", command, err)
	}

	return stdout, nil
}

func removeContainer(client *client.Client, containerId string) {
	err := client.ContainerRemove(context.Background(), containerId, types.ContainerRemoveOptions{})
	if err != nil {
		log.Printf("error removing container ID %s: %v", containerId, err)
	}
}

func getContainerLogs(client *client.Client, containerId string) (string, string, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}

	body, err := client.ContainerLogs(context.Background(), containerId, options)
	if err != nil {
		return "", "", fmt.Errorf("error getting logs: %w", err)
	}

	defer body.Close()

	var stdOutBuff, stdErrBuff bytes.Buffer
	_, err = stdcopy.StdCopy(&stdOutBuff, &stdErrBuff, body)
	if err != nil {
		return "", "", fmt.Errorf("error reading logs: %w", err)
	}

	return stdOutBuff.String(), stdErrBuff.String(), nil
}
