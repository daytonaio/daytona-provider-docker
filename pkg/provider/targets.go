package provider

import (
	"encoding/json"
	"os"
	"path"

	"github.com/daytonaio/daytona/pkg/provider"
)

func GetTargets(basePath string) (map[string]provider.ProviderTarget, error) {
	targetsFilePath := path.Join(basePath, "targets.json")

	file, err := os.Open(targetsFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	targets := map[string]provider.ProviderTarget{}
	err = json.NewDecoder(file).Decode(&targets)
	if err != nil {
		return nil, err
	}

	return targets, nil
}

func SetTargets(basePath string, targets map[string]provider.ProviderTarget) error {
	targetsFilePath := path.Join(basePath, "targets.json")

	file, err := os.Create(targetsFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	content, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		return err
	}

	_, err = file.Write(content)
	return err
}

func InitializeTargets(basePath string) error {
	targetsFilePath := path.Join(basePath, "targets.json")

	if _, err := os.Stat(targetsFilePath); os.IsNotExist(err) {
		defaultTargets := map[string]provider.ProviderTarget{
			"local": {
				Name:    "local",
				Options: "{\"Container Image\": \"daytonaio/workspace-project\"}",
			},
		}

		return SetTargets(basePath, defaultTargets)
	}

	return nil
}
