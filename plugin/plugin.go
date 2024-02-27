package plugin

import (
	"encoding/json"
	"errors"
	"os"
	"path"

	"provider/plugin/util"

	"github.com/daytonaio/daytona/pkg/provider"
	"github.com/daytonaio/daytona/pkg/types"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

type DockerProvider struct {
	BasePath          *string
	ServerDownloadUrl *string
	ServerVersion     *string
	ServerUrl         *string
	ServerApiUrl      *string
}

type workspaceMetadata struct {
	NetworkId string
}

func (p *DockerProvider) Initialize(req provider.InitializeProviderRequest) (*types.Empty, error) {
	p.BasePath = &req.BasePath
	p.ServerDownloadUrl = &req.ServerDownloadUrl
	p.ServerVersion = &req.ServerVersion
	p.ServerUrl = &req.ServerUrl
	p.ServerApiUrl = &req.ServerApiUrl
	return new(types.Empty), nil
}

func (p DockerProvider) GetInfo() (provider.ProviderInfo, error) {
	return provider.ProviderInfo{
		Name:    "docker-provider",
		Version: "0.0.1",
	}, nil
}

func (p DockerProvider) Configure() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (p DockerProvider) getProjectPath(basePath string, project *types.Project) string {
	return path.Join(basePath, "workspaces", project.WorkspaceId, "projects", project.Name)
}

func (p DockerProvider) CreateWorkspace(workspace *types.Workspace) (*types.Empty, error) {
	err := util.CreateNetwork(workspace.Id)
	return new(types.Empty), err
}

func (p DockerProvider) StartWorkspace(workspace *types.Workspace) (*types.Empty, error) {
	return new(types.Empty), nil
}

func (p DockerProvider) StopWorkspace(workspace *types.Workspace) (*types.Empty, error) {
	return new(types.Empty), nil
}

func (p DockerProvider) DestroyWorkspace(workspace *types.Workspace) (*types.Empty, error) {
	if p.BasePath == nil {
		return new(types.Empty), errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	err := os.RemoveAll(path.Join(*p.BasePath, "workspaces", workspace.Id))
	if err != nil {
		return new(types.Empty), err
	}

	return new(types.Empty), util.RemoveNetwork(workspace.Id)
}

func (p DockerProvider) GetWorkspaceInfo(workspace *types.Workspace) (*types.WorkspaceInfo, error) {
	providerMetadata, err := p.getWorkspaceMetadata(workspace)
	if err != nil {
		return nil, err
	}

	workspaceInfo := &types.WorkspaceInfo{
		Name:             workspace.Name,
		ProviderMetadata: providerMetadata,
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

func (p DockerProvider) CreateProject(project *types.Project) (*types.Empty, error) {
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

func (p DockerProvider) StartProject(project *types.Project) (*types.Empty, error) {
	err := util.StartContainer(project)
	return new(types.Empty), err
}

func (p DockerProvider) StopProject(project *types.Project) (*types.Empty, error) {
	err := util.StopContainer(project)
	return new(types.Empty), err
}

func (p DockerProvider) DestroyProject(project *types.Project) (*types.Empty, error) {
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

func (p DockerProvider) GetProjectInfo(project *types.Project) (*types.ProjectInfo, error) {
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
			Name:             project.Name,
			IsRunning:        isRunning,
			Created:          "",
			Started:          "",
			Finished:         "",
			ProviderMetadata: "{\"state\": \"container not found\"}",
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
		projectInfo.ProviderMetadata = string(metadata)
	} else {
		log.Warn("Could not get container labels for project: ", project.Name)
	}

	return projectInfo, nil
}

func (p DockerProvider) getWorkspaceMetadata(workspace *types.Workspace) (string, error) {
	metadata := workspaceMetadata{
		NetworkId: workspace.Id,
	}

	jsonContent, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	return string(jsonContent), nil
}
