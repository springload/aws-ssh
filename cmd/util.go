package cmd

import (
	"os"
	"path"
	"strings"

	"github.com/apex/log"
	"github.com/go-ini/ini"
	homedir "github.com/mitchellh/go-homedir"
)

// gets profiles. The current Go AWS SDK doesn't have this function, whereas python boto3 has it. Why?
func getProfiles() ([]string, error) {
	// use a map here to get rid of duplicates
	var profiles = make(map[string]struct{})

	home, err := homedir.Dir()
	if err != nil {
		return []string{}, err
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

		for _, section := range config.SectionStrings() {
			// skip the default section (top level keys/values)
			if section == ini.DefaultSection {
				continue
			}
			if strings.HasPrefix(section, sectionPrefix) {
				profiles[section[len(sectionPrefix):]] = struct{}{}
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
	var profilesList []string

	for profile := range profiles {
		profilesList = append(profilesList, profile)
	}
	return profilesList, nil
}
