package main

import (
	"github.com/daytonaio/daytona/plugins/provisioner"
	provisioner_manager "github.com/daytonaio/daytona/plugins/provisioner/manager"
	"github.com/hashicorp/go-plugin"

	provisioner_plugin "provisioner_plugin/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: provisioner_manager.ProvisionerHandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"docker-provisioner": &provisioner.ProvisionerPlugin{Impl: &provisioner_plugin.DockerProvisioner{}},
		},
	})
}
