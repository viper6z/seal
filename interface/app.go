package main

import (
	"go.yaml.in/yaml/v4"
	"io"
	"os"
	"fmt"
	"errors"
	"strings"
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

	err := loadApplication(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load application: %v\n", err)
		os.Exit(1)
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

//loadApplication opens, decodes, and closes an application manifest.
func loadApplication (path string) (error) {
	manifest, err := openApplicationFile(path)
	if err != nil {
		return err
	}
	defer manifest.Close()
	application, err := decodeApplicationYAML(manifest)
	if err != nil {
		return err
	}
	err = validateApplication(application)
	if err != nil {
		return err
	}
	return nil
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

//domain validation
func validateApplication(application Application) (error) {
	if application.Name == "" {
		return errors.New("Name can't be empty")
	}

	}
	if application.Image == "" || !strings.HasPrefix(application.Image, "ghcr.io") {
		return errors.New("Image needs to be non empty and in viper6z ghcr")
	}
	
	if application.InternalPort < 1 || application.InternalPort > 65535 {
		return errors.New("internal_port must be a number between 1 and 65535")
	}

	if application.ExposureType == "" || application.ExposureType != "public" && application.ExposureType != "internal" {
		return errors.New("exposure type must not be empty, exposure type is either public or internal")
	}
	
	if application.ExposureType == "internal" && len(application.AllowedPublicRoutes) > 0 {
		return errors.New("internal apps may not have public routes")
	}

	if application.ExposureType == "public" && len(application.AllowedPublicRoutes) == 0 {
		return errors.New("public apps need atleast 1 public route")
	}
	
	return nil
}
