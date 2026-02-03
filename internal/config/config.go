package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type File struct {
	Version int `yaml:"version,omitempty"`

	Authtoken  string `yaml:"authtoken,omitempty"`
	ServerAddr string `yaml:"server_addr,omitempty"`

	Inspect     *bool  `yaml:"inspect,omitempty"`
	InspectAddr string `yaml:"inspect_addr,omitempty"`
}

func DefaultPath() string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "eosrift", "eosrift.yml")
	}

	dir, err := os.UserConfigDir()
	if err == nil && dir != "" {
		return filepath.Join(dir, "eosrift", "eosrift.yml")
	}

	home := os.Getenv("HOME")
	if home == "" {
		return filepath.Join("eosrift.yml")
	}
	return filepath.Join(home, ".config", "eosrift", "eosrift.yml")
}

func Load(path string) (File, bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, false, nil
		}
		return File{}, false, err
	}

	var cfg File
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return File{}, true, err
	}
	return cfg, true, nil
}

func Save(path string, cfg File) error {
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if len(b) == 0 || b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}

	tmp, err := os.CreateTemp(dir, ".eosrift-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}

func ControlURLFromServerAddr(serverAddr string) (string, error) {
	s := strings.TrimSpace(serverAddr)
	if s == "" {
		return "", errors.New("server address is empty")
	}

	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil {
			return "", err
		}

		switch u.Scheme {
		case "ws", "wss":
			if u.Path == "" || u.Path == "/" {
				u.Path = "/control"
			}
			return u.String(), nil
		case "http", "https":
			wsScheme := "ws"
			if u.Scheme == "https" {
				wsScheme = "wss"
			}

			basePath := strings.TrimSuffix(u.Path, "/")
			return (&url.URL{
				Scheme: wsScheme,
				Host:   u.Host,
				Path:   basePath + "/control",
			}).String(), nil
		default:
			return "", fmt.Errorf("unsupported scheme: %q", u.Scheme)
		}
	}

	hostPart := s
	pathPart := ""
	if i := strings.Index(hostPart, "/"); i >= 0 {
		pathPart = hostPart[i:]
		hostPart = hostPart[:i]
	}
	hostPart = strings.TrimSpace(hostPart)
	if hostPart == "" {
		return "", errors.New("invalid server address")
	}

	scheme := "wss"
	if _, port, err := net.SplitHostPort(hostPart); err == nil {
		if port == "443" {
			scheme = "wss"
		} else {
			scheme = "ws"
		}
	}

	basePath := strings.TrimSuffix(pathPart, "/")
	return (&url.URL{
		Scheme: scheme,
		Host:   hostPart,
		Path:   basePath + "/control",
	}).String(), nil
}
