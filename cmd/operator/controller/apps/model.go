package apps

import "gopkg.in/yaml.v3"

type (
	AppValues struct {
		Config AppConfig `yaml:"homeCloud"`
	}
	AppConfig struct {
		Namespace   string
		Routes      []AppRoute
		Databases   []AppDatabase
		Persistence []AppPersistence
		Secrets     []AppSecret
		Disks       []AppDisk
	}
	AppRoute struct {
		Name    string
		Service AppService
	}
	AppService struct {
		Name string
		Port uint32
	}
	AppDatabase struct {
		Name string
		Type string
		Init string
	}
	AppPersistence struct {
		Name string
		Size string
	}
	AppSecret struct {
		Name string
		Keys []SecretKey
	}
	SecretKey struct {
		Name                string
		Length              int
		NoSpecialCharacters bool `yaml:"noSpecialCharacters"`
	}
	AppDisk struct {
		Name      string
		ClaimName string `yaml:"claimName"`
	}
)

// ToValues converts the AppConfig into a values map and adds to the given existing values map
func (c AppConfig) ToValues(values map[string]interface{}) (map[string]interface{}, error) {

	b, err := yaml.Marshal(AppValues{Config: c})
	if err != nil {
		return nil, err
	}

	v := make(map[string]interface{})
	err = yaml.Unmarshal(b, v)
	if err != nil {
		return nil, err
	}

	values["homeCloud"] = v["homeCloud"]

	return values, nil
}
