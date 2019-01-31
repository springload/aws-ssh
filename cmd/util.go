package cmd

import (
	"os"
	"path"
	"strings"

	"github.com/go-ini/ini"
	homedir "github.com/mitchellh/go-homedir"
)

// gets profiles. The current Go AWS SDK doesn't have this function, whereas python boto3 has it. Why?
func getProfiles() ([]string, error) {
	var profiles []string

	configFile := os.Getenv("AWS_CONFIG_FILE")
	if configFile == "" {
		home, err := homedir.Dir()
		if err != nil {
			return profiles, err
		}
		configFile = path.Join(home, ".aws", "config")
	}
	config, err := ini.Load(configFile)
	if err != nil {
		return profiles, err
	}
	for _, section := range config.SectionStrings() {
		if strings.HasPrefix(section, "profile ") {
			profiles = append(profiles, section[8:])
		}
	}

	return profiles, nil
}
