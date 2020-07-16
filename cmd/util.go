package cmd

import (
	"aws-ssh/lib"
	"os"
	"path"
	"strings"

	"github.com/apex/log"
	"github.com/go-ini/ini"
	homedir "github.com/mitchellh/go-homedir"
)

// gets profiles. The current Go AWS SDK doesn't have this function, whereas python boto3 has it. Why?
func getProfiles() ([]lib.ProfileConfig, error) {
	// use a map here to get rid of duplicates
	var profiles = make(map[string]lib.ProfileConfig)

	home, err := homedir.Dir()
	if err != nil {
		return []lib.ProfileConfig{}, err
	}

	loadFile := func(envVariableName, defaultFileName, sectionPrefix string) error {
		configFile := os.Getenv(envVariableName)
		if configFile == "" {
			configFile = defaultFileName
		}
		config, err := ini.Load(configFile)
		if err != nil {
			return err
		}

		for _, section := range config.Sections() {
			name := section.Name()
			// skip the default section (top level keys/values)
			if name == ini.DefaultSection {
				continue
			}
			if strings.HasPrefix(name, sectionPrefix) {
				name = name[len(sectionPrefix):]
				if _, exists := profiles[name]; !exists {
					config := lib.ProfileConfig{Name: name}
					if section.HasKey("aws-ssh-domain") {
						config.Domain = section.Key("aws-ssh-domain").Value()
					}
					log.Debugf("Got profile - %s", name)
					profiles[name] = config
				} else {
					log.Debugf("Skipping duplicate profile - %s", name)
				}
			}
		}
		return nil
	}

	if err := loadFile("AWS_CONFIG_FILE", path.Join(home, ".aws", "config"), "profile "); err != nil {
		log.WithError(err).Warn("Couldn't load the shared config file")
	}
	if err := loadFile("AWS_SHARED_CREDENTIALS_FILE", path.Join(home, ".aws", "credentials"), ""); err != nil {
		log.WithError(err).Warn("Couldn't load the shared credentials file")
	}

	// convert the map to a list
	profilesList := make([]lib.ProfileConfig, 0, len(profiles))

	for _, profile := range profiles {
		profilesList = append(profilesList, profile)
	}
	return profilesList, nil
}

func contains(slice []string, element string) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}
