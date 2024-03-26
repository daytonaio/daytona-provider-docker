package util

import (
	"context"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

func PullImage(client *client.Client, imageName string, logWriter *io.Writer) error {
	ctx := context.Background()

	tag := "latest"
	tagSplit := strings.Split(imageName, ":")
	if len(tagSplit) == 2 {
		tag = tagSplit[1]
	}

	if tag != "latest" {
		images, err := client.ImageList(ctx, types.ImageListOptions{
			Filters: filters.NewArgs(filters.Arg("reference", imageName)),
		})
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

		if found {
			if logWriter != nil {
				(*logWriter).Write([]byte("Image found locally\n"))
			}
			return nil
		}
	}

	if logWriter != nil {
		(*logWriter).Write([]byte("Pulling image...\n"))
	}
	responseBody, err := client.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer responseBody.Close()
	_, err = io.Copy(io.Discard, responseBody)
	if err != nil {
		return err
	}

	if logWriter != nil {
		(*logWriter).Write([]byte("Image pulled successfully\n"))
	}

	return nil
}
