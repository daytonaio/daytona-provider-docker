package types

import (
	"encoding/json"

	"github.com/daytonaio/daytona/pkg/provider"
)

type TargetConfigOptions struct {
	RemoteHostname   *string `json:"Remote Hostname,omitempty"`
	RemotePort       *int    `json:"Remote Port,omitempty"`
	RemoteUser       *string `json:"Remote User,omitempty"`
	RemotePassword   *string `json:"Remote Password,omitempty"`
	RemotePrivateKey *string `json:"Remote Private Key Path,omitempty"`
	SockPath         *string `json:"Sock Path,omitempty"`
	TargetDataDir    *string `json:"Target Data Dir,omitempty"`
}

func GetTargetManifest() *provider.TargetConfigManifest {
	return &provider.TargetConfigManifest{
		"Remote Hostname": provider.TargetConfigProperty{
			Type:              provider.TargetConfigPropertyTypeString,
			DisabledPredicate: "^local$",
		},
		"Remote Port": provider.TargetConfigProperty{
			Type:              provider.TargetConfigPropertyTypeInt,
			DefaultValue:      "22",
			DisabledPredicate: "^local$",
		},
		"Remote User": provider.TargetConfigProperty{
			Type: provider.TargetConfigPropertyTypeString,
			// TODO: Add docs entry
			Description:       "Note: non-root user required",
			DisabledPredicate: "^local$",
		},
		"Remote Password": provider.TargetConfigProperty{
			Type:              provider.TargetConfigPropertyTypeString,
			DisabledPredicate: "^local$",
			InputMasked:       true,
		},
		"Remote Private Key Path": provider.TargetConfigProperty{
			Type:              provider.TargetConfigPropertyTypeFilePath,
			DefaultValue:      "~/.ssh",
			DisabledPredicate: "^local$",
		},
		"Sock Path": provider.TargetConfigProperty{
			Type:         provider.TargetConfigPropertyTypeString,
			DefaultValue: "/var/run/docker.sock",
		},
		"Target Data Dir": provider.TargetConfigProperty{
			Type:              provider.TargetConfigPropertyTypeString,
			DefaultValue:      "/tmp/daytona-data",
			Description:       "The directory on the remote host where the target data will be stored",
			DisabledPredicate: "^local$",
		},
	}
}

func ParseTargetConfigOptions(optionsJson string) (opts *TargetConfigOptions, isLocal bool, err error) {
	var targetOptions TargetConfigOptions
	err = json.Unmarshal([]byte(optionsJson), &targetOptions)
	if err != nil {
		return nil, false, err
	}

	return &targetOptions, targetOptions.RemoteHostname == nil, nil
}
