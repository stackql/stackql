package providerconfig

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

func ReadProviderConfig(filePath string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	err = yaml.Unmarshal(data, &m)
	return m, err
}
