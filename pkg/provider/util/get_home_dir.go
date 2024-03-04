package util

import (
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func GetHomeDirectory(client *client.Client, containerId string, user string) (*string, error) {
	result, err := ExecSync(client, containerId, types.ExecConfig{
		Cmd:  []string{"/bin/sh", "-c", "cd && pwd"},
		User: user,
	}, nil)
	if err != nil {
		return nil, err
	}

	homeDir := strings.Trim(result.StdOut, "\n")

	return &homeDir, nil
}
