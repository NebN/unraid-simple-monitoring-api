package conf

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Conf struct {
	Networks     []string            `yaml:"networks"`
	Disks        map[string][]string `yaml:"disks"`
	Units        Units               `yaml:"units"`
	LoggingLevel string              `yaml:"loggingLevel"`
	CpuTemp      *string             `yaml:"cpuTemp"`
	Include      []string            `yaml:"include"`
	Exclude      []string            `yaml:"exclude"`
	Cors         *Cors               `yaml:"cors"`
}

type Cors struct {
	Origin  string `yaml:"origin"`
	Methods string `yaml:"methods"`
	Headers string `yaml:"headers"`
}

type Units struct {
	Array  string `yaml:"array"`
	Cache  string `yaml:"cache"`
	Pools  string `yaml:"pools"`
	Memory string `yaml:"memory"`
}

func rawConf(path string) (Conf, error) {
	conf := Conf{}
	content, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(content, &conf)
	if err != nil {
		return conf, err
	}

	return conf, nil
}

var (
	defaultUnits = Units{
		Array:  "Gi",
		Cache:  "Gi",
		Pools:  "Gi",
		Memory: "Mi",
	}
)

func applyDefaults(conf Conf) Conf {
	if len(conf.Units.Array) == 0 {
		conf.Units.Array = defaultUnits.Array
	}
	if len(conf.Units.Cache) == 0 {
		conf.Units.Cache = defaultUnits.Cache
	}
	if len(conf.Units.Pools) == 0 {
		conf.Units.Pools = defaultUnits.Pools
	}
	if len(conf.Units.Memory) == 0 {
		conf.Units.Memory = defaultUnits.Memory
	}
	return conf
}

func ReadConf(path string) (Conf, error) {
	conf, err := rawConf(path)
	if err != nil {
		return conf, err
	}
	return applyDefaults(conf), nil
}
