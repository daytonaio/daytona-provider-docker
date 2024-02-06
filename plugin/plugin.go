package plugin

import (
	"context"
	"errors"
	"os"
	"path"

	"provisioner_plugin/plugin/util"

	"github.com/daytonaio/daytona/grpc/proto/types"
	"github.com/daytonaio/daytona/grpc/utils"
	"github.com/daytonaio/daytona/plugin/provisioner"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

type DockerProvisioner struct {
	provisioner.Provisioner
	BasePath string
}

func (p DockerProvisioner) GetName() string {
	return "docker"
}

func (p DockerProvisioner) GetVersion() string {
	return "0.0.1"
}

func (p DockerProvisioner) Configure() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p DockerProvisioner) SetConfig(config interface{}) error {
	return errors.New("not implemented")
}

func (p DockerProvisioner) getProjectPath(project *types.Project) string {
	return path.Join(p.BasePath, "workspaces", project.WorkspaceId, "projects", project.Name)
}

func (p DockerProvisioner) CreateWorkspace(workspace *types.Workspace) error {
	return nil
}

func (p DockerProvisioner) StartWorkspace(workspace *types.Workspace) error {
	return nil
}

func (p DockerProvisioner) StopWorkspace(workspace *types.Workspace) error {
	return nil
}

func (p DockerProvisioner) DestroyWorkspace(workspace *types.Workspace) error {
	return nil
}

func (p DockerProvisioner) GetWorkspaceMetadata(workspace *types.Workspace) (*interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p DockerProvisioner) CreateProject(project *types.Project) error {
	log.Info("Initializing project: ", project.Name)

	clonePath := p.getProjectPath(project)

	err := os.MkdirAll(clonePath, 0755)
	if err != nil {
		return err
	}

	err = util.CloneRepository(project, clonePath)
	if err != nil {
		return err
	}

	// TODO: Project image from config
	err = util.InitContainer(project, clonePath, "daytonaio/workspace-project")
	if err != nil {
		return err
	}

	err = util.StartContainer(project)
	if err != nil {
		return err
	}

	return nil
}

func (p DockerProvisioner) StartProject(project *types.Project) error {
	return util.StartContainer(project)
}

func (p DockerProvisioner) StopProject(project *types.Project) error {
	return util.StopContainer(project)
}

func (p DockerProvisioner) DestroyProject(project *types.Project) error {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	err = cli.ContainerRemove(ctx, util.GetContainerName(project), docker_types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil && !client.IsErrNotFound(err) {
		return err
	}

	err = cli.VolumeRemove(ctx, util.GetVolumeName(project), true)
	if err != nil && !client.IsErrNotFound(err) {
		return err
	}

	err = os.RemoveAll(p.getProjectPath(project))
	if err != nil {
		return err
	}

	return nil
}

func (p DockerProvisioner) GetProjectInfo(project *types.Project) (*types.ProjectInfo, error) {
	isRunning := true
	info, err := util.GetContainerInfo(project)
	if err != nil {
		if client.IsErrNotFound(err) {
			log.Debug("Container not found, project is not running")
			isRunning = false
		} else {
			return nil, err
		}
	}

	provisionerMetadata, err := utils.StructToProtobufStruct(info.Config.Labels)
	if err != nil {
		return nil, err
	}

	return &types.ProjectInfo{
		Name:                project.Name,
		IsRunning:           isRunning,
		Created:             info.Created,
		Started:             info.State.StartedAt,
		Finished:            info.State.FinishedAt,
		ProvisionerMetadata: provisionerMetadata,
	}, nil
}
