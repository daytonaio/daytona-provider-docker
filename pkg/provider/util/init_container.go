package util

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/daytonaio/daytona/pkg/types"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
)

func InitContainer(client *client.Client, project *types.Project, workdirPath, imageName, serverDownloadUrl, serverVersion, serverUrl, serverApiUrl string) error {
	ctx := context.Background()

	images, err := client.ImageList(ctx, docker_types.ImageListOptions{})
	if err != nil {
		return err
	}

	found := false
	for _, image := range images {
		for _, tag := range image.RepoTags {
			if strings.HasPrefix(tag, imageName) {
				found = true
				break
			}
		}
	}

	if !found {
		log.Info("Image not found, pulling...")
		responseBody, err := client.ImagePull(ctx, imageName, docker_types.ImagePullOptions{})
		if err != nil {
			return err
		}
		defer responseBody.Close()
		_, err = io.Copy(io.Discard, responseBody)
		if err != nil {
			return err
		}
		log.Info("Image pulled successfully")
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: GetVolumeName(project),
			Target: "/var/lib/docker",
		},
	}

	envVars := []string{
		"DAYTONA_WS_ID=" + project.WorkspaceId,
		"DAYTONA_WS_DIR=" + path.Join("/workspaces", project.Name),
		"DAYTONA_WS_PROJECT_NAME=" + project.Name,
		"DAYTONA_WS_PROJECT_REPOSITORY_URL=" + project.Repository.Url,
		"DAYTONA_SERVER_API_KEY=" + project.ApiKey,
		"DAYTONA_SERVER_VERSION=" + serverVersion,
		"DAYTONA_SERVER_URL=" + serverUrl,
		"DAYTONA_SERVER_API_URL=" + serverApiUrl,
	}

	_, err = client.ContainerCreate(ctx, &container.Config{
		Hostname: project.Name,
		Image:    imageName,
		Labels: map[string]string{
			"daytona.workspace.id":                     project.WorkspaceId,
			"daytona.workspace.project.name":           project.Name,
			"daytona.workspace.project.repository.url": project.Repository.Url,
			// todo: Add more properties here
		},
		User: "daytona",
		Env:  envVars,
		Cmd:  []string{"bash", "-c", fmt.Sprintf("curl %s | bash && daytona agent", serverDownloadUrl)},
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

func WaitForBinaryDownload(client *client.Client, project *types.Project) error {
	ctx := context.Background()

	for {
		_, err := client.ContainerStatPath(ctx, GetContainerName(project), "/usr/local/bin/daytona")

		if err == nil {
			break
		}
	}

	return nil
}