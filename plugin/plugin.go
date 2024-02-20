package plugin

import (
	"errors"
	"os"
	"path"

	"provisioner_plugin/plugin/util"

	"github.com/daytonaio/daytona/common/types"
	"github.com/daytonaio/daytona/plugins/provisioner"
	"github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

type DockerProvisioner struct {
	BasePath          *string
	ServerDownloadUrl *string
	ServerVersion     *string
	ServerUrl         *string
	ServerApiUrl      *string
}

type workspaceMetadata struct {
	NetworkId string
}

func (p *DockerProvisioner) Initialize(req provisioner.InitializeProvisionerRequest) error {
	p.BasePath = &req.BasePath
	p.ServerDownloadUrl = &req.ServerDownloadUrl
	p.ServerVersion = &req.ServerVersion
	p.ServerUrl = &req.ServerUrl
	p.ServerApiUrl = &req.ServerApiUrl
	return nil
}

func (p DockerProvisioner) GetInfo() (provisioner.ProvisionerInfo, error) {
	return provisioner.ProvisionerInfo{
		Name:    "docker-provisioner",
		Version: "0.0.1",
	}, nil
}

func (p DockerProvisioner) Configure() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p DockerProvisioner) SetConfig(config interface{}) error {
	return errors.New("not implemented")
}

func (p DockerProvisioner) getProjectPath(basePath string, project *types.Project) string {
	return path.Join(basePath, "workspaces", project.WorkspaceId, "projects", project.Name)
}

func (p DockerProvisioner) CreateWorkspace(workspace *types.Workspace) error {
	return util.CreateNetwork(workspace.Id)
}

func (p DockerProvisioner) StartWorkspace(workspace *types.Workspace) error {
	return nil
}

func (p DockerProvisioner) StopWorkspace(workspace *types.Workspace) error {
	return nil
}

func (p DockerProvisioner) DestroyWorkspace(workspace *types.Workspace) error {
	if p.BasePath == nil {
		return errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	err := os.RemoveAll(path.Join(*p.BasePath, "workspaces", workspace.Id))
	if err != nil {
		return err
	}

	return util.RemoveNetwork(workspace.Id)
}

func (p DockerProvisioner) GetWorkspaceInfo(workspace *types.Workspace) (*types.WorkspaceInfo, error) {
	provisionerMetadata, err := p.getWorkspaceMetadata(workspace)
	if err != nil {
		return nil, err
	}

	workspaceInfo := &types.WorkspaceInfo{
		Name:                workspace.Name,
		ProvisionerMetadata: &provisionerMetadata,
	}

	projectInfos := []*types.ProjectInfo{}
	for _, project := range workspace.Projects {
		projectInfo, err := p.GetProjectInfo(project)
		if err != nil {
			return nil, err
		}
		projectInfos = append(projectInfos, projectInfo)
	}
	workspaceInfo.Projects = projectInfos

	return workspaceInfo, nil
}

func (p DockerProvisioner) CreateProject(project *types.Project) error {
	log.Info("Initializing project: ", project.Name)

	if p.ServerDownloadUrl == nil {
		return errors.New("ServerDownloadUrl not set. Did you forget to call Initialize?")
	}

	if p.BasePath == nil {
		return errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	if p.ServerUrl == nil {
		return errors.New("ServerUrl not set. Did you forget to call Initialize?")
	}

	if p.ServerApiUrl == nil {
		return errors.New("ServerApiUrl not set. Did you forget to call Initialize?")
	}

	serverVersion := "latest"
	if p.ServerVersion != nil {
		serverVersion = *p.ServerVersion
	}

	clonePath := p.getProjectPath(*p.BasePath, project)

	err := os.MkdirAll(clonePath, 0755)
	if err != nil {
		return err
	}

	// TODO: Project image from config
	err = util.InitContainer(project, clonePath, "daytonaio/workspace-project", *p.ServerDownloadUrl, serverVersion, *p.ServerUrl, *p.ServerApiUrl)
	if err != nil {
		return err
	}

	err = util.StartContainer(project)
	if err != nil {
		return err
	}

	err = util.SetGitConfig(project, "daytona")
	if err != nil {
		return err
	}

	err = util.CloneRepository(project, path.Join("/workspaces", project.Name))
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
	err := util.RemoveContainer(project)
	if err != nil {
		return err
	}

	if p.BasePath == nil {
		return errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	err = os.RemoveAll(p.getProjectPath(*p.BasePath, project))
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

	if info != nil || info.State == nil {
		return nil, errors.New("could not get container state")
	}

	projectInfo := &types.ProjectInfo{
		Name:      project.Name,
		IsRunning: isRunning,
		Created:   info.Created,
		Started:   info.State.StartedAt,
		Finished:  info.State.FinishedAt,
	}

	if info.Config != nil && info.Config.Labels != nil {
		var metadata *interface{}
		mapstructure.Decode(info.Config.Labels, &metadata)
		projectInfo.ProvisionerMetadata = metadata
	} else {
		log.Warn("Could not get container labels for project: ", project.Name)
	}

	return projectInfo, nil
}

func (p DockerProvisioner) getWorkspaceMetadata(workspace *types.Workspace) (interface{}, error) {
	return workspaceMetadata{
		NetworkId: workspace.Id,
	}, nil
}
