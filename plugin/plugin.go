package plugin

import (
	"errors"
	"os"
	"path"

	"provisioner_plugin/plugin/util"

	"github.com/daytonaio/daytona/grpc/proto/types"
	"github.com/daytonaio/daytona/grpc/utils"
	"github.com/daytonaio/daytona/plugin/provisioner/grpc/proto"
	"github.com/docker/docker/client"
	structpb "github.com/golang/protobuf/ptypes/struct"
	log "github.com/sirupsen/logrus"
)

type DockerProvisioner struct {
	BasePath *string
}

type workspaceMetadata struct {
	NetworkId string
}

func (p *DockerProvisioner) Initialize(req *proto.InitializeProvisionerRequest) error {
	p.BasePath = &req.BasePath
	return nil
}

func (p DockerProvisioner) GetInfo() (*proto.ProvisionerInfo, error) {
	return &proto.ProvisionerInfo{
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
	err := os.RemoveAll(path.Join("/tmp", "workspaces", workspace.Id))
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

func (p DockerProvisioner) CreateProject(project *types.Project) error {
	log.Info("Initializing project: ", project.Name)

	if p.BasePath == nil {
		return errors.New("BasePath not set. Did you forget to call Initialize?")
	}

	clonePath := p.getProjectPath(*p.BasePath, project)

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

	var provisionerMetadata *structpb.Struct = nil

	if info != nil && info.Config != nil && info.Config.Labels != nil {
		provisionerMetadata, err = utils.StructToProtobufStruct(info.Config.Labels)
		if err != nil {
			return nil, err
		}
	} else {
		log.Warn("Could not get container labels for project: ", project.Name)
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

func (p DockerProvisioner) getWorkspaceMetadata(workspace *types.Workspace) (*structpb.Struct, error) {
	metadata := workspaceMetadata{
		NetworkId: workspace.Id,
	}

	protoMetadata, err := utils.StructToProtobufStruct(metadata)
	if err != nil {
		return nil, err
	}
	return protoMetadata, nil
}
