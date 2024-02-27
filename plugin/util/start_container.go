package util

import (
	"context"
	"time"

	"github.com/daytonaio/daytona/pkg/types"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func StartContainer(project *types.Project) error {
	containerName := GetContainerName(project)
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	err = cli.ContainerStart(ctx, containerName, docker_types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	// make sure container is running
	//	TODO: timeout
	for {
		inspect, err := cli.ContainerInspect(ctx, containerName)
		if err != nil {
			return err
		}

		if inspect.State.Running {
			break
		}

		time.Sleep(1 * time.Second)
	}

	// start dockerd
	execConfig := docker_types.ExecConfig{
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"dockerd"},
		User:         "root",
	}
	execResp, err := cli.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		return err
	}

	err = cli.ContainerExecStart(ctx, execResp.ID, docker_types.ExecStartCheck{})
	if err != nil {
		return err
	}

	//	todo: wait for dockerd to start
	time.Sleep(3 * time.Second)

	return nil
}
