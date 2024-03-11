package plugin

import (
	"encoding/json"
	"errors"
	"os"
	"path"

	"provisioner_plugin/plugin/util"

	"github.com/daytonaio/daytona/common/types"
	"github.com/daytonaio/daytona/plugins/provisioner"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
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

func (p *DockerProvisioner) Initialize(req provisioner.InitializeProvisionerRequest) (*types.Empty, error) {
	p.BasePath = &req.BasePath
	p.ServerDownloadUrl = &req.ServerDownloadUrl
	p.ServerVersion = &req.ServerVersion
	p.ServerUrl = &req.ServerUrl
	p.ServerApiUrl = &req.ServerApiUrl
	return new(types.Empty), nil
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

func (p DockerProvisioner) getProjectPath(basePath string, project *types.Project) string {
	return path.Join(basePath, "workspaces", project.WorkspaceId, "projects", project.Name)
}

func (p DockerProvisioner) CreateWorkspace(workspace *types.Workspace) (*types.Empty, error) {
	err := util.CreateNetwork(workspace.Id)
	return new(types.Empty), err
}

func (p DockerProvisioner) StartWorkspace(workspace *types.Workspace) (*types.Empty, error) {
	return new(types.Empty), nil
}

func (p DockerProvisioner) StopWorkspace(workspace *types.Workspace) (*types.Empty, error) {
	return new(types.Empty), nil
}

func (p DockerProvisioner) DestroyWorkspace(workspace *types.Workspace) (*types.Empty, error) {
	if p.BasePath == nil {
		return new(types.Empty), errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	err := os.RemoveAll(path.Join(*p.BasePath, "workspaces", workspace.Id))
	if err != nil {
		return new(types.Empty), err
	}

	return new(types.Empty), util.RemoveNetwork(workspace.Id)
}

func (p DockerProvisioner) GetWorkspaceInfo(workspace *types.Workspace) (*types.WorkspaceInfo, error) {
	provisionerMetadata, err := p.getWorkspaceMetadata(workspace)
	if err != nil {
		return nil, err
	}

	workspaceInfo := &types.WorkspaceInfo{
		Name:                workspace.Name,
		ProvisionerMetadata: provisionerMetadata,
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

func (p DockerProvisioner) CreateProject(project *types.Project) (*types.Empty, error) {
	log.Info("Initializing project: ", project.Name)

	if p.ServerDownloadUrl == nil {
		return new(types.Empty), errors.New("ServerDownloadUrl not set. Did you forget to call Initialize?")
	}

	if p.BasePath == nil {
		return new(types.Empty), errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	if p.ServerUrl == nil {
		return new(types.Empty), errors.New("ServerUrl not set. Did you forget to call Initialize?")
	}

	if p.ServerApiUrl == nil {
		return new(types.Empty), errors.New("ServerApiUrl not set. Did you forget to call Initialize?")
	}

	serverVersion := "latest"
	if p.ServerVersion != nil {
		serverVersion = *p.ServerVersion
	}

	clonePath := p.getProjectPath(*p.BasePath, project)

	err := os.MkdirAll(clonePath, 0755)
	if err != nil {
		return new(types.Empty), err
	}

	// TODO: Project image from config
	err = util.InitContainer(project, clonePath, "daytonaio/workspace-project", *p.ServerDownloadUrl, serverVersion, *p.ServerUrl, *p.ServerApiUrl)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.StartContainer(project)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.SetGitConfig(project, "daytona")
	if err != nil {
		return new(types.Empty), err
	}

	err = util.WaitForBinaryDownload(project)
	if err != nil {
		return new(types.Empty), err
	}

	_, err = util.ExecSync(util.GetContainerName(project), docker_types.ExecConfig{
		User:       "daytona",
		Privileged: true,
		Cmd:        []string{"sudo", "chown", "-R", "daytona:daytona", "/workspaces"},
	}, nil)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.CloneRepository(project, path.Join("/workspaces", project.Name))
	if err != nil {
		return new(types.Empty), err
	}

	return new(types.Empty), nil
}

func (p DockerProvisioner) StartProject(project *types.Project) (*types.Empty, error) {
	err := util.StartContainer(project)
	return new(types.Empty), err
}

func (p DockerProvisioner) StopProject(project *types.Project) (*types.Empty, error) {
	err := util.StopContainer(project)
	return new(types.Empty), err
}

func (p DockerProvisioner) DestroyProject(project *types.Project) (*types.Empty, error) {
	err := util.RemoveContainer(project)
	if err != nil {
		return new(types.Empty), err
	}

	if p.BasePath == nil {
		return new(types.Empty), errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	err = os.RemoveAll(p.getProjectPath(*p.BasePath, project))
	if err != nil {
		return new(types.Empty), err
	}

	return new(types.Empty), nil
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

	if info == nil || info.State == nil {
		return &types.ProjectInfo{
			Name:                project.Name,
			IsRunning:           isRunning,
			Created:             "",
			Started:             "",
			Finished:            "",
			ProvisionerMetadata: "{\"state\": \"container not found\"}",
		}, nil
	}

	projectInfo := &types.ProjectInfo{
		Name:      project.Name,
		IsRunning: isRunning,
		Created:   info.Created,
		Started:   info.State.StartedAt,
		Finished:  info.State.FinishedAt,
	}

	if info.Config != nil && info.Config.Labels != nil {
		metadata, err := json.Marshal(info.Config.Labels)
		if err != nil {
			return nil, err
		}
		projectInfo.ProvisionerMetadata = string(metadata)
	} else {
		log.Warn("Could not get container labels for project: ", project.Name)
	}

	return projectInfo, nil
}

func (p DockerProvisioner) getWorkspaceMetadata(workspace *types.Workspace) (string, error) {
	metadata := workspaceMetadata{
		NetworkId: workspace.Id,
	}

	jsonContent, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	return string(jsonContent), nil
}
