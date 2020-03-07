package config

import (
	"crypto/sha256"
	"crypto/tls"
	"io/ioutil"

	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// Config contains the server (the webhook) cert and key.
type Config struct {
	CertFile string
	KeyFile  string
}

// Sidecar is the configuration for sidecars
type Sidecar struct {
	Containers []v1.Container `yaml:"containers"`
	Volumes    []v1.Volume    `yaml:"volumes"`
}

func configTLS(config Config) *tls.Config {
	sCert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		klog.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}
}

// LoadFromYAML loads configuration from the given YAML
func LoadFromYAML(file string, out interface{}) error {
	data, err := ioutil.ReadFile(file)

	if err != nil {
		return err
	}

	klog.V(2).Infof("New configuration: sha256sum", sha256.Sum256(data))

	if err := yaml.Unmarshal(data, out); err != nil {
		return err
	}

	return nil
}
