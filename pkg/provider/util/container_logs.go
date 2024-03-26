package util

import (
	"context"
	"io"

	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func GetContainerLogs(client *client.Client, containerName string, logWriter *io.Writer) error {
	if logWriter == nil {
		return nil
	}

	logs, err := client.ContainerLogs(context.Background(), containerName, docker_types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return err
	}
	defer logs.Close()

	_, err = stdcopy.StdCopy(*logWriter, *logWriter, logs)

	return err
}
