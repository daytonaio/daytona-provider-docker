package provider_test

import (
	"context"
	"encoding/json"
	"provider/pkg/client"
	"provider/pkg/provider/util"
	"testing"

	"github.com/daytonaio/daytona/pkg/provider"
	"github.com/daytonaio/daytona/pkg/types"

	docker_provider "provider/pkg/provider"
	provider_types "provider/pkg/types"

	docker_types "github.com/docker/docker/api/types"
	docker_client "github.com/docker/docker/client"
)

var dockerProvider = &docker_provider.DockerProvider{}
var targetOptions = &provider_types.TargetOptions{
	ContainerImage: "daytonaio/workspace-project",
}
var sockDir = "/tmp/target-socks"
var optionsString string

var project1 = &types.Project{
	Name: "test",
	Repository: &types.Repository{
		Id:     "123",
		Url:    "https://github.com/daytonaio/daytona",
		Name:   "daytona",
		Branch: "main",
	},
	WorkspaceId: "123",
}

var workspace = &types.Workspace{
	Id:     "123",
	Name:   "test",
	Target: "local",
	Projects: []*types.Project{
		project1,
	},
}

func TestCreateWorkspace(t *testing.T) {
	wsReq := &provider.WorkspaceRequest{
		TargetOptions: optionsString,
		Workspace:     workspace,
	}

	_, err := dockerProvider.CreateWorkspace(wsReq)
	if err != nil {
		t.Errorf("Error creating workspace: %s", err)
	}

	_, err = getDockerClient().NetworkInspect(context.Background(), workspace.Id, docker_types.NetworkInspectOptions{})
	if err != nil {
		t.Errorf("Expected network to exist")
	}
}

func TestGetWorkspaceInfo(t *testing.T) {
	wsReq := &provider.WorkspaceRequest{
		TargetOptions: optionsString,
		Workspace:     workspace,
	}

	workspaceInfo, err := dockerProvider.GetWorkspaceInfo(wsReq)
	if err != nil || workspaceInfo == nil {
		t.Errorf("Error getting workspace info: %s", err)
	}

	var workspaceMetadata provider_types.WorkspaceMetadata
	err = json.Unmarshal([]byte(workspaceInfo.ProviderMetadata), &workspaceMetadata)
	if err != nil {
		t.Errorf("Error unmarshalling workspace metadata: %s", err)
	}

	if workspaceMetadata.NetworkId != wsReq.Workspace.Id {
		t.Errorf("Expected network id %s, got %s", wsReq.Workspace.Id, workspaceMetadata.NetworkId)
	}
}

func TestDestroyWorkspace(t *testing.T) {
	wsReq := &provider.WorkspaceRequest{
		TargetOptions: optionsString,
		Workspace:     workspace,
	}

	_, err := dockerProvider.DestroyWorkspace(wsReq)
	if err != nil {
		t.Errorf("Error deleting workspace: %s", err)
	}

	dockerClient, err := client.GetClient(*targetOptions, sockDir)
	if err != nil {
		t.Errorf("Error getting docker client: %s", err)
	}

	_, err = dockerClient.NetworkInspect(context.Background(), workspace.Id, docker_types.NetworkInspectOptions{})
	if err == nil {
		t.Errorf("Expected network to not exist")
	}
}

func TestCreateProject(t *testing.T) {
	TestCreateWorkspace(t)

	projectReq := &provider.ProjectRequest{
		TargetOptions: optionsString,
		Project:       project1,
	}

	_, err := dockerProvider.CreateProject(projectReq)
	if err != nil {
		t.Errorf("Error creating project: %s", err)
	}

	_, err = getDockerClient().ContainerInspect(context.Background(), util.GetContainerName(project1))
	if err != nil {
		t.Errorf("Expected container to exist")
	}
}

func TestDestroyProject(t *testing.T) {
	projectReq := &provider.ProjectRequest{
		TargetOptions: optionsString,
		Project:       project1,
	}

	_, err := dockerProvider.DestroyProject(projectReq)
	if err != nil {
		t.Errorf("Error deleting project: %s", err)
	}

	_, err = getDockerClient().ContainerInspect(context.Background(), util.GetContainerName(project1))
	if err == nil {
		t.Errorf("Expected container to not exist")
	}

	TestDestroyWorkspace(t)
}

func getDockerClient() *docker_client.Client {
	dockerClient, err := client.GetClient(*targetOptions, sockDir)
	if err != nil {
		panic(err)
	}

	return dockerClient
}

func init() {
	_, err := dockerProvider.Initialize(provider.InitializeProviderRequest{
		BasePath:          "/tmp/workspaces",
		ServerDownloadUrl: "https://download.daytona.io/daytona/get-server.sh",
		ServerVersion:     "latest",
		ServerUrl:         "",
		ServerApiUrl:      "",
	})
	if err != nil {
		panic(err)
	}

	opts, err := json.Marshal(targetOptions)
	if err != nil {
		panic(err)
	}

	optionsString = string(opts)
}