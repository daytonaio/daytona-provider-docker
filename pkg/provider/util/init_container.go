package util

import (
	"context"
	"fmt"
	"path"

	"github.com/daytonaio/daytona/pkg/provider/util"
	"github.com/daytonaio/daytona/pkg/workspace"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

func InitContainer(client *client.Client, project *workspace.Project, workdirPath, imageName, serverDownloadUrl string) error {
	ctx := context.Background()

	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: GetVolumeName(project),
			Target: "/var/lib/docker",
		},
	}

	envVars := []string{
		"DAYTONA_WS_DIR=" + path.Join("/workspaces", project.Name),
	}

	for key, value := range project.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}

	_, err := client.ContainerCreate(ctx, &container.Config{
		Hostname: project.Name,
		Image:    imageName,
		Labels: map[string]string{
			"daytona.workspace.id":                     project.WorkspaceId,
			"daytona.workspace.project.name":           project.Name,
			"daytona.workspace.project.repository.url": project.Repository.Url,
			// todo: Add more properties here
		},
		User:         "daytona",
		Env:          envVars,
		Cmd:          []string{"bash", "-c", util.GetProjectStartScript(serverDownloadUrl, project.ApiKey)},
		AttachStdout: true,
		AttachStderr: true,
	}, &container.HostConfig{
		Privileged: true,
		Binds: []string{
			fmt.Sprintf("%s:/workspaces", workdirPath),
			"/tmp/daytona:/tmp/daytona",
		},
		Mounts:      mounts,
		NetworkMode: container.NetworkMode(project.WorkspaceId),
	}, nil, nil, GetContainerName(project)) //	TODO: namespaced names
	if err != nil {
		return err
	}

	return nil
}

func WaitForBinaryDownload(client *client.Client, project *workspace.Project) error {
	ctx := context.Background()

	for {
		_, err := client.ContainerStatPath(ctx, GetContainerName(project), "/usr/local/bin/daytona")

		if err == nil {
			break
		}
	}

	return nil
}
