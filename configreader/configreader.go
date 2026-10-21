package configreader

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type (
	ConfigEntity struct {
		CompanyName string
		HTTP        ServerHttp
		Root        RootUser
	}

	ServerHttp struct {
		Port       string
		ServerIP   string
		ServerDNSN string
	}

	RootUser struct {
		Email, Password string
	}
)

func CheckConfigFile(path string) (*os.File, error) {
	descriptor, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			descriptor, err = os.Create(path)
			if err != nil {
				return nil, fmt.Errorf("config file creating error: %w", err)
			}
			cfg := ConfigEntity{
				HTTP: ServerHttp{Port: "443"},
				Root: RootUser{Email: "example@example.com", Password: "asdlkdfjiopejkddsapjp1225234412"},
			}
			yamlEncoder := yaml.NewEncoder(descriptor)
			yamlEncoder.SetIndent(2)

			if err := yamlEncoder.Encode(cfg); err != nil {
				os.Remove(path)
				return nil, fmt.Errorf("config file creating error: %w", err)
			}

			if err := yamlEncoder.Close(); err != nil { // явно, а не defer
				os.Remove(path)
				return nil, fmt.Errorf("config file flush error: %w", err)
			}

			if _, err := descriptor.Seek(0, io.SeekStart); err != nil {
				return nil, fmt.Errorf("config file seek error: %w", err)
			}

			log.Println("The configuration file has been created with default data — success")
		} else {
			return nil, fmt.Errorf("config file with path %s - searching error:%w", path, err)
		}
	}

	return descriptor, err
}

func Parse(f *os.File) (ConfigEntity, error) {
	cfg := ConfigEntity{}
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return ConfigEntity{}, fmt.Errorf("config file parsing error: %w", err)
	}
	return cfg, nil
}
