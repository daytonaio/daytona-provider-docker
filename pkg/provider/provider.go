package provider

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"provider/pkg/client"
	"provider/pkg/provider/util"
	provider_types "provider/pkg/types"

	"github.com/daytonaio/daytona/pkg/provider"
	"github.com/daytonaio/daytona/pkg/types"
	docker_types "github.com/docker/docker/api/types"
	docker_client "github.com/docker/docker/client"
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

	return new(types.Empty), InitializeTargets(*p.BasePath)
}

func (p DockerProvider) GetInfo() (provider.ProviderInfo, error) {
	return provider.ProviderInfo{
		Name:    "docker-provider",
		Version: "0.0.1",
	}, nil
}

func (p DockerProvider) GetTargets() (*[]provider.ProviderTarget, error) {
	targets, err := GetTargets(*p.BasePath)
	if err != nil {
		return nil, err
	}

	list := []provider.ProviderTarget{}
	for _, target := range targets {
		list = append(list, target)
	}
	return &list, nil
}

func (p DockerProvider) GetTargetManifest() (*provider.ProviderTargetManifest, error) {
	return provider_types.GetTargetManifest(), nil
}

func (p DockerProvider) SetTarget(target provider.ProviderTarget) (*types.Empty, error) {
	targets, err := GetTargets(*p.BasePath)
	if err != nil {
		return nil, err
	}
	targets[target.Name] = target
	SetTargets(*p.BasePath, targets)
	return new(types.Empty), nil
}

func (p DockerProvider) RemoveTarget(name string) (*types.Empty, error) {
	if name == "local" {
		return new(types.Empty), errors.New("cannot remove 'local' target")
	}
	targets, err := GetTargets(*p.BasePath)
	if err != nil {
		return nil, err
	}
	delete(targets, name)
	SetTargets(*p.BasePath, targets)
	return new(types.Empty), nil
}

func (p DockerProvider) getProjectPath(basePath string, project *types.Project) string {
	return path.Join(basePath, "workspaces", project.WorkspaceId, "projects", project.Name)
}

func (p DockerProvider) CreateWorkspace(workspace *types.Workspace) (*types.Empty, error) {
	client, err := p.getClient(workspace.ProviderTarget.Target)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.CreateNetwork(client, workspace.Id)
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

	client, err := p.getClient(workspace.ProviderTarget.Target)
	if err != nil {
		return new(types.Empty), err
	}

	return new(types.Empty), util.RemoveNetwork(client, workspace.Id)
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

	targets, err := GetTargets(*p.BasePath)
	if err != nil {
		return nil, err
	}

	target, ok := targets[project.ProviderTarget.Target]
	if !ok {
		return nil, errors.New("target not found")
	}
	targetOptions, err := provider_types.GetTargetOptions(target)
	if err != nil {
		return new(types.Empty), err
	}

	client, err := p.getClient(project.ProviderTarget.Target)
	if err != nil {
		return new(types.Empty), err
	}

	clonePath := p.getProjectPath(*p.BasePath, project)

	err = os.MkdirAll(clonePath, 0755)
	if err != nil {
		return new(types.Empty), err
	}

	// TODO: Project image from config
	err = util.InitContainer(client, project, clonePath, targetOptions.ContainerImage, *p.ServerDownloadUrl, serverVersion, *p.ServerUrl, *p.ServerApiUrl)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.StartContainer(client, project)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.SetGitConfig(client, project, "daytona")
	if err != nil {
		return new(types.Empty), err
	}

	err = util.WaitForBinaryDownload(client, project)
	if err != nil {
		return new(types.Empty), err
	}

	_, err = util.ExecSync(client, util.GetContainerName(project), docker_types.ExecConfig{
		User:       "daytona",
		Privileged: true,
		Cmd:        []string{"sudo", "chown", "-R", "daytona:daytona", "/workspaces"},
	}, nil)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.CloneRepository(client, project, path.Join("/workspaces", project.Name))
	if err != nil {
		return new(types.Empty), err
	}

	return new(types.Empty), nil
}

func (p DockerProvider) StartProject(project *types.Project) (*types.Empty, error) {
	client, err := p.getClient(project.ProviderTarget.Target)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.StartContainer(client, project)
	return new(types.Empty), err
}

func (p DockerProvider) StopProject(project *types.Project) (*types.Empty, error) {
	client, err := p.getClient(project.ProviderTarget.Target)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.StopContainer(client, project)
	return new(types.Empty), err
}

func (p DockerProvider) DestroyProject(project *types.Project) (*types.Empty, error) {
	client, err := p.getClient(project.ProviderTarget.Target)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.RemoveContainer(client, project)
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
	client, err := p.getClient(project.ProviderTarget.Target)
	if err != nil {
		return nil, err
	}

	isRunning := true
	info, err := util.GetContainerInfo(client, project)
	if err != nil {
		if docker_client.IsErrNotFound(err) {
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

func (p DockerProvider) getClient(targetName string) (*docker_client.Client, error) {
	if p.BasePath == nil {
		return nil, errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	targets, err := GetTargets(*p.BasePath)
	if err != nil {
		return nil, err
	}

	target, ok := targets[targetName]
	if !ok {
		return nil, errors.New("target not found")
	}

	return client.GetClient(target, *p.BasePath)
}
