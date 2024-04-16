package util

import (
	"context"
	"io"
	"time"

	"github.com/daytonaio/daytona/pkg/workspace"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func StartContainer(client *client.Client, project *workspace.Project, logWriter *io.Writer) error {
	containerName := GetContainerName(project)
	ctx := context.Background()

	inspect, err := client.ContainerInspect(ctx, containerName)

	if err == nil && inspect.State.Running {
		return nil
	}

	err = client.ContainerStart(ctx, containerName, docker_types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	// make sure container is running
	//	TODO: timeout
	for {
		inspect, err := client.ContainerInspect(ctx, containerName)
		if err != nil {
			return err
		}

		if inspect.State.Running {
			break
		}

		time.Sleep(1 * time.Second)
	}

	if logWriter != nil {
		go func() {
			err := GetContainerLogs(client, containerName, logWriter)
			if err != nil {
				(*logWriter).Write([]byte(err.Error()))
			}
		}()
	}

	// start dockerd
	execConfig := docker_types.ExecConfig{
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"dockerd"},
		User:         "root",
	}
	execResp, err := client.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		return err
	}

	err = client.ContainerExecStart(ctx, execResp.ID, docker_types.ExecStartCheck{})
	if err != nil {
		return err
	}

	//	todo: wait for dockerd to start
	time.Sleep(3 * time.Second)

	return nil
}
