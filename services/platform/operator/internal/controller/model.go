package controller

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
	}
	AppRoute struct {
		Name    string
		Service AppService
	}
	AppService struct {
		Name string
		Port int
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
)
