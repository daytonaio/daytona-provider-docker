package util

import (
	"github.com/daytonaio/daytona/pkg/types"
	docker_types "github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
)

func CloneRepository(project *types.Project, clonePath string) error {
	repo := project.Repository

	log.WithFields(log.Fields{
		"project": project.Name,
	}).Info("Cloning repository: " + repo.Url)

	_, err := ExecSync(GetContainerName(project), docker_types.ExecConfig{
		User: "daytona",
		Cmd:  []string{"git", "clone", repo.Url, clonePath},
	}, nil)

	return err
}
