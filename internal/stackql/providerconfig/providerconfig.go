package providerconfig

import (
	"os"

	"gopkg.in/yaml.v2"
)

func ReadProviderConfig(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	err = yaml.Unmarshal(data, &m)
	return m, err
}
