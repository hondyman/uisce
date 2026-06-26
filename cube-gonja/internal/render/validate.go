package render

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type cubeDoc struct {
	Cubes []struct {
		Name       string `yaml:"name"`
		DataSource string `yaml:"data_source"`
	} `yaml:"cubes"`
	Views []struct {
		Name       string `yaml:"name"`
		DataSource string `yaml:"data_source"`
	} `yaml:"views"`
}

func ValidateYAMLHardBinding(path string, data []byte, allowed map[string]struct{}) error {
	var doc cubeDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%s: YAML parse error: %w", filepath.Base(path), err)
	}
	check := func(kind, name, ds string) error {
		if ds == "" {
			return fmt.Errorf("%s %q missing data_source", kind, name)
		}
		if _, ok := allowed[ds]; !ok {
			return fmt.Errorf("%s %q has invalid data_source %q", kind, name, ds)
		}
		return nil
	}
	for _, c := range doc.Cubes {
		if err := check("cube", c.Name, c.DataSource); err != nil {
			return err
		}
	}
	for _, v := range doc.Views {
		if err := check("view", v.Name, v.DataSource); err != nil {
			return err
		}
	}
	return nil
}
