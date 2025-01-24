package types

import (
	"encoding/json"

	"github.com/daytonaio/daytona/pkg/models"
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

func GetTargetConfigManifest() *models.TargetConfigManifest {
	return &models.TargetConfigManifest{
		"Remote Hostname": models.TargetConfigProperty{
			Type:              models.TargetConfigPropertyTypeString,
			DisabledPredicate: "^local$",
		},
		"Remote Port": models.TargetConfigProperty{
			Type:              models.TargetConfigPropertyTypeInt,
			DefaultValue:      "22",
			DisabledPredicate: "^local$",
		},
		"Remote User": models.TargetConfigProperty{
			Type: models.TargetConfigPropertyTypeString,
			// TODO: Add docs entry
			Description:       "Note: non-root user required",
			DisabledPredicate: "^local$",
		},
		"Remote Password": models.TargetConfigProperty{
			Type:              models.TargetConfigPropertyTypeString,
			DisabledPredicate: "^local$",
			InputMasked:       true,
		},
		"Remote Private Key Path": models.TargetConfigProperty{
			Type:              models.TargetConfigPropertyTypeFilePath,
			DefaultValue:      "~/.ssh",
			DisabledPredicate: "^local$",
		},
		"Sock Path": models.TargetConfigProperty{
			Type:         models.TargetConfigPropertyTypeString,
			DefaultValue: "/var/run/docker.sock",
		},
		"Target Data Dir": models.TargetConfigProperty{
			Type:              models.TargetConfigPropertyTypeString,
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
