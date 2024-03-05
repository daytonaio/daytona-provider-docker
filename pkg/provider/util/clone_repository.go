package util

import (
	"github.com/daytonaio/daytona/pkg/types"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func CloneRepository(client *client.Client, project *types.Project, clonePath string) error {
	repo := project.Repository

	log.WithFields(log.Fields{
		"project": project.Name,
	}).Info("Cloning repository: " + repo.Url)

	cloneCmd := []string{"git", "clone", repo.Url, clonePath}

	if repo.Branch != "" && repo.Branch != repo.Sha {
		cloneCmd = append(cloneCmd, "-b", repo.Branch, "--single-branch")
	}

	_, err := ExecSync(client, GetContainerName(project), docker_types.ExecConfig{
		User: "daytona",
		Cmd:  cloneCmd,
	}, nil)

	if repo.Sha != "" && repo.Branch == repo.Sha {
		_, err = ExecSync(client, GetContainerName(project), docker_types.ExecConfig{
			User: "daytona",
			Cmd:  []string{"git", "checkout", repo.Sha},
		}, nil)
	}

	return err
}
