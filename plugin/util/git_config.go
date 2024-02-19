package util

import (
	"bytes"
	"path"

	"github.com/daytonaio/daytona/common/grpc/proto/types"
	"gopkg.in/ini.v1"
)

// TODO: Move this to the daytona repo and import it from there
func SetGitConfig(project *types.Project, user string) error {
	containerId := GetContainerName(project)

	homePath, err := GetHomeDirectory(containerId, user)
	if err != nil {
		return err
	}

	gitConfigFileName := path.Join(*homePath, ".gitconfig")

	var gitConfigContent []byte
	gitConfigContent, err = ReadFile(containerId, user, gitConfigFileName)
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

	cfg.Section("credential").NewKey("helper", "daytona git-cred")

	var buf bytes.Buffer
	_, err = cfg.WriteTo(&buf)
	if err != nil {
		return err
	}

	err = WriteFile(containerId, user, gitConfigFileName, buf.String())

	if err != nil {
		return err
	}

	return nil
}
