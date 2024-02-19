package util

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"io"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
)

func FileExists(containerId string, user string, dstFilePath string) (bool, error) {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		return false, err
	}

	execConfig := types.ExecConfig{
		Cmd:  []string{"stat", dstFilePath},
		User: user,
	}

	resp, err := cli.ContainerExecCreate(context.Background(), containerId, execConfig)
	if err != nil {
		return false, err
	}

	attachResp, err := cli.ContainerExecAttach(context.Background(), resp.ID, types.ExecStartCheck{})
	if err != nil {
		return false, nil
	}
	defer attachResp.Close()

	return true, nil
}

func ReadFile(containerId string, user string, dstFilePath string) ([]byte, error) {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Open a connection to the container
	reader, _, err := cli.CopyFromContainer(ctx, containerId, dstFilePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Extract the file content from the tar archive
	tarReader := tar.NewReader(reader)
	var content bytes.Buffer

	for {
		_, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		// Read the content from the tar entry
		if _, err := io.Copy(&content, tarReader); err != nil {
			return nil, err
		}
	}

	log.Debug(".gitconfig read file", content.String())

	return content.Bytes(), nil
}

func WriteFile(containerId string, user string, dstFilePath string, content string) error {
	dir := filepath.Dir(dstFilePath)

	_, err := ExecSync(containerId, types.ExecConfig{
		Cmd:  []string{"mkdir", "-p", dir},
		User: user,
	}, nil)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	var files = []struct {
		Name, Body string
	}{
		{filepath.Base(dstFilePath), content},
	}
	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			log.Fatal(err)
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			log.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		log.Fatal(err)
	}

	cli, err := client.NewClientWithOpts()
	if err != nil {
		return err
	}

	err = cli.CopyToContainer(context.Background(), containerId, dir, bufio.NewReader(&buf), types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		return err
	}

	_, err = ExecSync(containerId, types.ExecConfig{
		Cmd:  []string{"chown", user + ":" + user, dstFilePath},
		User: "root",
	}, nil)

	return err
}

func Remove(containerId string, user string, path string) {
	ExecSync(containerId, types.ExecConfig{
		Cmd:  []string{"rm", "-rf", path},
		User: user,
	}, nil)
}
