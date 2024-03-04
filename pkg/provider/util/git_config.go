package util

import (
	"bytes"
	"path"

	"github.com/daytonaio/daytona/pkg/types"
	"github.com/docker/docker/client"
	"gopkg.in/ini.v1"
)

// TODO: Move this to the daytona repo and import it from there
func SetGitConfig(client *client.Client, project *types.Project, user string) error {
	containerId := GetContainerName(project)

	homePath, err := GetHomeDirectory(client, containerId, user)
	if err != nil {
		return err
	}

	gitConfigFileName := path.Join(*homePath, ".gitconfig")

	var gitConfigContent []byte
	gitConfigContent, err = ReadFile(client, containerId, user, gitConfigFileName)
	if err != nil {
		gitConfigContent = []byte{}
	}

	cfg, err := ini.Load(gitConfigContent)
	if err != nil {
		return err
	}

	if !cfg.HasSection("credential") {
		cfg.NewSection("credential")
	}

	cfg.Section("credential").NewKey("helper", "/usr/local/bin/daytona git-cred")

	if project.Repository != nil && project.Repository.GitUserData != nil {
		if !cfg.HasSection("user") {
			cfg.NewSection("user")
		}

		cfg.Section("user").NewKey("name", project.Repository.GitUserData.Name)
		cfg.Section("user").NewKey("email", project.Repository.GitUserData.Email)
	}

	var buf bytes.Buffer
	_, err = cfg.WriteTo(&buf)
	if err != nil {
		return err
	}

	err = WriteFile(client, containerId, user, gitConfigFileName, buf.String())

	if err != nil {
		return err
	}

	return nil
}
