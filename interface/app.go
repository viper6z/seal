package main

import (
	"go.yaml.in/yaml/v4"
	"io"
	"os"
)

func main() {

	if len(os.Args) != 3 {
		os.Stderr.WriteString("Invalid input, the format is seal validate <path>\n")
		os.Exit(2)
	}

	if os.Args[1] != "validate" {
		os.Stderr.WriteString("Invalid input, allowed subcommands are: validate\n")
		os.Exit(2)
	}
	
	result, err := openApplicationFile(os.Args[2])
		if err != nil {
		os.Stderr.WriteString("Invalid path\n")
		os.Exit(2)
	}
	defer result.Close() //this waits until main is done to close the file
	
	_, err = decodeApplicationYAML(result)

	if err != nil {
		os.Stderr.WriteString("error decoding yaml manifest")
		os.Exit(2)
	}
}

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
