package main

import (
	"errors"
	"fmt"
	"go.yaml.in/yaml/v4"
	"io"
	"os"
	"strings"
)

func main() {

	if len(os.Args) != 3 {
		os.Stderr.WriteString("Invalid input, the format is seal validate/deploy <path>\n")
		os.Exit(2)
	}

	if os.Args[1] != "validate" && os.Args[1] != "deploy" {
		os.Stderr.WriteString("Invalid input, allowed subcommands are: validate/deploy\n")
		os.Exit(2)
	}

	application, err := loadApplication(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load application: %v\n", err)
		os.Exit(1)
	}

	if os.Args[1] == "deploy" {
		compose, err := loadCompose()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load compose file: %v\n", err)
			os.Exit(1)
		}

		err = deploymentPreCheck(application, compose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Precheck failed: %v\n", err)
			os.Exit(1)
		}
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

// loadApplication opens, decodes, and closes an application manifest.
func loadApplication(path string) (Application, error) {
	manifest, err := openApplicationFile(path)
	if err != nil {
		return Application{}, err
	}
	defer manifest.Close()
	application, err := decodeApplicationYAML(manifest)
	if err != nil {
		return Application{}, err
	}
	err = validateApplication(application)
	if err != nil {
		return Application{}, err
	}
	return application, nil
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

// domain validation
func validateApplication(application Application) error {
	if application.Name == "" {
		return errors.New("Name can't be empty")
	}

	if application.Image == "" || !strings.HasPrefix(application.Image, "ghcr.io/") {
		return errors.New("Image needs to be non empty and in viper6z ghcr")
	}

	for _, rune := range application.Name {
		if !(rune >= 'a' && rune <= 'z') && !(rune >= '0' && rune <= '9') && !(rune == '-') {
			return errors.New("name must only include letters in a-z, numbers in 0-9, and -")
		}
	}

	if application.Name[0] == '-' || application.Name[len(application.Name)-1] == '-' {
		return errors.New("no leading or trailing hyphens allowed in name")
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

	for _, s := range application.AllowedPublicRoutes {
		if !(strings.HasPrefix(s, "/")) {
			return errors.New("all public routes must start with /")
		}
	}

	m := make(map[string]bool)
	for _, s := range application.AllowedPublicRoutes {
		_, ok := m[s]
		if ok == false {
			m[s] = true
		} else {
			return errors.New("no duplicate routes allowed")
		}
	}

	return nil
}

// struct representing compose file
type Compose struct {
	Services map[string]yaml.Node `yaml:"services"`
	Networks map[string]yaml.Node `yaml:"networks"`
}

// open compose.yaml, works only from seal root
func tryOpenCompose() (*os.File, error) {
	file, err := os.Open("compose.yaml")
	if err != nil {
		return nil, err
	}
	return file, nil
}

func decodeComposeYAML(reader io.Reader) (Compose, error) {
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)

	var compose Compose

	err := decoder.Decode(&compose)

	if err != nil {
		return compose, err
	}

	return compose, err
}

func loadCompose() (Compose, error) {
	file, err := tryOpenCompose()
	if err != nil {
		return Compose{}, err
	}
	defer file.Close()
	compose, err := decodeComposeYAML(file)
	if err != nil {
		return Compose{}, err
	}
	return compose, nil
}

// now when we have both the application struct and the compose struct, we will make sure the service name is not already in use
func deploymentPreCheck(application Application, compose Compose) error {
	_, ok := compose.Services[application.Name]
	if ok {
		return errors.New("service name already taken")
	}
	_, ok = compose.Networks["backend"]
	if !ok {
		return errors.New("no backend network exists in the docker compose configuration")
	}
	return nil
}

type Service struct {
	Image string `yaml:"image"`
	Networks []string `yaml:"networks"`
}
//this function takes an application and uses some of its data to make a service
func renderService(application Application) (service Service) {
	service.Image = application.Image
	service.Networks = []string{"backend"}
	return service
}

func encodeService(service Service) error {
	node := yaml.Node{}
}
