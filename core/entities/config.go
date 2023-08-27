package entities

type Config struct {
	LogFilePath string              `yaml:"logFilePath"`
	Reporters   []map[string]string `yaml:"reporters"`
	Workflows   map[string]string   `yaml:"workflows"`
}
