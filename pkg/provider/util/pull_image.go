package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type pullEvent struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	Error          string `json:"error,omitempty"`
	Progress       string `json:"progress,omitempty"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
}

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
	// _, err = io.Copy(io.Discard, responseBody)
	// if err != nil {
	// 	return err
	// }

	cursor := Cursor{
		logWriter: *logWriter,
	}
	layers := make([]string, 0)
	oldIndex := len(layers)

	var event *pullEvent
	decoder := json.NewDecoder(responseBody)

	for {
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		imageID := event.ID

		// Check if the line is one of the final two ones
		if strings.HasPrefix(event.Status, "Digest:") || strings.HasPrefix(event.Status, "Status:") {
			(*logWriter).Write([]byte(fmt.Sprintf("%s\n", event.Status)))
			continue
		}

		// Check if ID has already passed once
		index := 0
		for i, v := range layers {
			if v == imageID {
				index = i + 1
				break
			}
		}

		// Move the cursor
		if index > 0 {
			diff := index - oldIndex

			if diff > 1 {
				down := diff - 1
				cursor.moveDown(down)
			} else if diff < 1 {
				up := diff*(-1) + 1
				cursor.moveUp(up)
			}

			oldIndex = index
		} else {
			layers = append(layers, event.ID)
			diff := len(layers) - oldIndex

			if diff > 1 {
				cursor.moveDown(diff) // Return to the last row
			}

			oldIndex = len(layers)
		}

		// cursor.clearLine()

		if event.Status == "Pull complete" {
			(*logWriter).Write([]byte(fmt.Sprintf("%s: %s\n", event.ID, event.Status)))
		} else {
			(*logWriter).Write([]byte(fmt.Sprintf("%s: %s %s\n", event.ID, event.Status, event.Progress)))
		}

	}

	if logWriter != nil {
		(*logWriter).Write([]byte("Image pulled successfully\n"))
	}

	return nil
}

// Cursor structure that implements some methods
// for manipulating command line's cursor
type Cursor struct {
	logWriter io.Writer
}

func (c *Cursor) moveUp(rows int) {
	c.logWriter.Write([]byte(fmt.Sprintf("\033[%dF", rows)))
}

func (c *Cursor) moveDown(rows int) {
	c.logWriter.Write([]byte(fmt.Sprintf("\033[%dE", rows)))
}

func (c *Cursor) clearLine() {
	c.logWriter.Write([]byte("\033[2K"))
}
