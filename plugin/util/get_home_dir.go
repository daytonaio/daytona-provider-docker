package util

import (
	"strings"

	"github.com/docker/docker/api/types"
)

func GetHomeDirectory(containerId string, user string) (*string, error) {
	result, err := ExecSync(containerId, types.ExecConfig{
		Cmd:  []string{"/bin/sh", "-c", "cd && pwd"},
		User: user,
	}, nil)
	if err != nil {
		return nil, err
	}

	homeDir := strings.Trim(result.StdOut, "\n")

	return &homeDir, nil
}
