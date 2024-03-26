package provider

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"runtime"

	internal "github.com/daytonaio/daytona-docker-provider/internal"
	log_writers "github.com/daytonaio/daytona-docker-provider/internal/log"
	"github.com/daytonaio/daytona-docker-provider/pkg/client"
	"github.com/daytonaio/daytona-docker-provider/pkg/provider/util"
	provider_types "github.com/daytonaio/daytona-docker-provider/pkg/types"

	"github.com/daytonaio/daytona/pkg/logger"
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
	LogsDir           *string
	RemoteSockDir     string
}

func (p *DockerProvider) Initialize(req provider.InitializeProviderRequest) (*types.Empty, error) {
	tmpDir := "/tmp"
	if runtime.GOOS == "windows" {
		tmpDir = os.TempDir()
		if tmpDir == "" {
			return new(types.Empty), errors.New("could not determine temp dir")
		}
	}

	p.RemoteSockDir = path.Join(tmpDir, "target-socks")

	// Clear old sockets
	err := os.RemoveAll(p.RemoteSockDir)
	if err != nil {
		return new(types.Empty), err
	}
	err = os.MkdirAll(p.RemoteSockDir, 0755)
	if err != nil {
		return new(types.Empty), err
	}

	p.BasePath = &req.BasePath
	p.ServerDownloadUrl = &req.ServerDownloadUrl
	p.ServerVersion = &req.ServerVersion
	p.ServerUrl = &req.ServerUrl
	p.ServerApiUrl = &req.ServerApiUrl
	p.LogsDir = &req.LogsDir

	return new(types.Empty), nil
}

func (p DockerProvider) GetInfo() (provider.ProviderInfo, error) {
	return provider.ProviderInfo{
		Name:    "docker-provider",
		Version: internal.Version,
	}, nil
}

func (p DockerProvider) GetTargetManifest() (*provider.ProviderTargetManifest, error) {
	return provider_types.GetTargetManifest(), nil
}

func (p DockerProvider) GetDefaultTargets() (*[]provider.ProviderTarget, error) {
	info, err := p.GetInfo()
	if err != nil {
		return nil, err
	}

	defaultTargets := []provider.ProviderTarget{
		{
			Name:         "local",
			ProviderInfo: info,
			Options:      "{\n\t\"Container Image\": \"daytonaio/workspace-project\",\n\t\"Sock Path\": \"/var/run/docker.sock\"\n}",
		},
	}
	return &defaultTargets, nil
}

func (p DockerProvider) getProjectPath(basePath string, project *types.Project) string {
	return path.Join(basePath, "workspaces", project.WorkspaceId, "projects", project.Name)
}

func (p DockerProvider) CreateWorkspace(workspaceReq *provider.WorkspaceRequest) (*types.Empty, error) {
	client, err := p.getClient(workspaceReq.TargetOptions)
	if err != nil {
		return new(types.Empty), err
	}

	targetOptions, err := provider_types.ParseTargetOptions(workspaceReq.TargetOptions)
	if err != nil {
		return new(types.Empty), err
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		wsLogWriter := logger.GetWorkspaceLogger(*p.LogsDir, workspaceReq.Workspace.Id)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, wsLogWriter)
		defer wsLogWriter.Close()
	}

	err = util.PullImage(client, targetOptions.ContainerImage, &logWriter)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.CreateNetwork(client, workspaceReq.Workspace.Id, &logWriter)
	return new(types.Empty), err
}

func (p DockerProvider) StartWorkspace(workspaceReq *provider.WorkspaceRequest) (*types.Empty, error) {
	return new(types.Empty), nil
}

func (p DockerProvider) StopWorkspace(workspaceReq *provider.WorkspaceRequest) (*types.Empty, error) {
	return new(types.Empty), nil
}

func (p DockerProvider) DestroyWorkspace(workspaceReq *provider.WorkspaceRequest) (*types.Empty, error) {
	if p.BasePath == nil {
		return new(types.Empty), errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	err := os.RemoveAll(path.Join(*p.BasePath, "workspaces", workspaceReq.Workspace.Id))
	if err != nil {
		return new(types.Empty), err
	}

	client, err := p.getClient(workspaceReq.TargetOptions)
	if err != nil {
		return new(types.Empty), err
	}

	return new(types.Empty), util.RemoveNetwork(client, workspaceReq.Workspace.Id)
}

func (p DockerProvider) GetWorkspaceInfo(workspaceReq *provider.WorkspaceRequest) (*types.WorkspaceInfo, error) {
	providerMetadata, err := p.getWorkspaceMetadata(workspaceReq)
	if err != nil {
		return nil, err
	}

	workspaceInfo := &types.WorkspaceInfo{
		Name:             workspaceReq.Workspace.Name,
		ProviderMetadata: providerMetadata,
	}

	projectInfos := []*types.ProjectInfo{}
	for _, project := range workspaceReq.Workspace.Projects {
		projectInfo, err := p.GetProjectInfo(&provider.ProjectRequest{
			TargetOptions: workspaceReq.TargetOptions,
			Project:       project,
		})
		if err != nil {
			return nil, err
		}
		projectInfos = append(projectInfos, projectInfo)
	}
	workspaceInfo.Projects = projectInfos

	return workspaceInfo, nil
}

func (p DockerProvider) CreateProject(projectReq *provider.ProjectRequest) (*types.Empty, error) {
	targetOptions, err := provider_types.ParseTargetOptions(projectReq.TargetOptions)
	if err != nil {
		return new(types.Empty), err
	}

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

	client, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(types.Empty), err
	}

	clonePath := p.getProjectPath(*p.BasePath, projectReq.Project)

	err = os.MkdirAll(clonePath, 0755)
	if err != nil {
		return new(types.Empty), err
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		wsLogWriter := logger.GetWorkspaceLogger(*p.LogsDir, projectReq.Project.WorkspaceId)
		projectLogWriter := logger.GetProjectLogger(*p.LogsDir, projectReq.Project.WorkspaceId, projectReq.Project.Name)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, wsLogWriter, projectLogWriter)
		defer wsLogWriter.Close()
		defer projectLogWriter.Close()
	}

	err = util.InitContainer(client, projectReq.Project, clonePath, targetOptions.ContainerImage, *p.ServerDownloadUrl)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.StartContainer(client, projectReq.Project, &logWriter)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.WaitForBinaryDownload(client, projectReq.Project)
	if err != nil {
		return new(types.Empty), err
	}

	_, err = util.ExecSync(client, util.GetContainerName(projectReq.Project), docker_types.ExecConfig{
		User:       "daytona",
		Privileged: true,
		Cmd:        []string{"sudo", "chown", "-R", "daytona:daytona", "/workspaces"},
	}, nil)
	if err != nil {
		return new(types.Empty), err
	}

	return new(types.Empty), nil
}

func (p DockerProvider) StartProject(projectReq *provider.ProjectRequest) (*types.Empty, error) {
	client, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(types.Empty), err
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.LogsDir != nil {
		wsLogWriter := logger.GetWorkspaceLogger(*p.LogsDir, projectReq.Project.WorkspaceId)
		projectLogWriter := logger.GetProjectLogger(*p.LogsDir, projectReq.Project.WorkspaceId, projectReq.Project.Name)
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, wsLogWriter, projectLogWriter)
		defer wsLogWriter.Close()
		defer projectLogWriter.Close()
	}

	err = util.StartContainer(client, projectReq.Project, &logWriter)
	return new(types.Empty), err
}

func (p DockerProvider) StopProject(projectReq *provider.ProjectRequest) (*types.Empty, error) {
	client, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.StopContainer(client, projectReq.Project)
	return new(types.Empty), err
}

func (p DockerProvider) DestroyProject(projectReq *provider.ProjectRequest) (*types.Empty, error) {
	client, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return new(types.Empty), err
	}

	err = util.RemoveContainer(client, projectReq.Project)
	if err != nil {
		return new(types.Empty), err
	}

	if p.BasePath == nil {
		return new(types.Empty), errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	err = os.RemoveAll(p.getProjectPath(*p.BasePath, projectReq.Project))
	if err != nil {
		return new(types.Empty), err
	}

	return new(types.Empty), nil
}

func (p DockerProvider) GetProjectInfo(projectReq *provider.ProjectRequest) (*types.ProjectInfo, error) {
	client, err := p.getClient(projectReq.TargetOptions)
	if err != nil {
		return nil, err
	}

	isRunning := true
	info, err := util.GetContainerInfo(client, projectReq.Project)
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
			Name:             projectReq.Project.Name,
			IsRunning:        isRunning,
			Created:          "",
			Started:          "",
			Finished:         "",
			ProviderMetadata: "{\"state\": \"container not found\"}",
		}, nil
	}

	projectInfo := &types.ProjectInfo{
		Name:      projectReq.Project.Name,
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
		log.Warn("Could not get container labels for project: ", projectReq.Project.Name)
	}

	return projectInfo, nil
}

func (p DockerProvider) getWorkspaceMetadata(workspaceReq *provider.WorkspaceRequest) (string, error) {
	metadata := provider_types.WorkspaceMetadata{
		NetworkId: workspaceReq.Workspace.Id,
	}

	jsonContent, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	return string(jsonContent), nil
}

func (p DockerProvider) getClient(targetOptionsJson string) (*docker_client.Client, error) {
	targetOptions, err := provider_types.ParseTargetOptions(targetOptionsJson)
	if err != nil {
		return nil, err
	}

	return client.GetClient(*targetOptions, p.RemoteSockDir)
}
