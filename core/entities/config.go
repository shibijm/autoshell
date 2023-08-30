package entities

type Config struct {
	Protected bool              `yaml:"protected"`
	Workflows map[string]string `yaml:"workflows"`
}
