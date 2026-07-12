package main

import (
	"go.yaml.in/yaml/v4"
	"io"
	"os"
)

// struct representing an application
type Application struct {
	Name                string   `yaml:"name"`
	Image               string   `yaml:"image"`
	InternalPort        int      `yaml:"internal_port"`
	ExposureType        string   `yaml:"exposure_type"`
	AllowedPublicRoutes []string `yaml:"allowed_public_routes"`
}

// receive path from arg
func openApplicationFile(path string) (*os.File, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	return file, err
}

// decode opened application file from yaml
func decodeApplicationYAML(reader io.Reader) (Application, error) {
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)

	var application Application

	err := decoder.Decode(&application)

	if err != nil {
		return application, err
	}

	return application, err
}
