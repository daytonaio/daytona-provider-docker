package main

import (
	"github.com/daytonaio/daytona/plugin/provisioner"
	provisioner_manager "github.com/daytonaio/daytona/plugin/provisioner/manager"
	"github.com/hashicorp/go-plugin"

	. "provisioner_plugin/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: provisioner_manager.ProvisionerHandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"docker-provisioner": &provisioner.ProvisionerPlugin{Impl: &DockerProvisioner{}},
		},
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
