package entities

type Config struct {
	Reporting []map[string]string `yaml:"reporting"`
	Workflows map[string]string   `yaml:"workflows"`
}
