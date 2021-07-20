package lib

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ProfileConfig represents an entry in aws config
type ProfileConfig struct {
	Name, // aws profile name
	Region, // region
	Domain string // domain if set with "aws-ssh-domain" in the config
}

type SSHEntries []SSHEntry

// SaveConfig saves the ssh config for the entries
func (e SSHEntries) SaveConfig(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("can't open file for writing: %s", err)
	}
	defer file.Close()
	for _, entry := range e {
		if _, err := io.WriteString(file, entry.ConfigFormat()); err != nil {
			return fmt.Errorf("can't write to file: %s", err)
		}
	}
	return nil
}

// SSHEntry represents an entry in ssh config
type SSHEntry struct {
	ProfileConfig ProfileConfig `yaml:"profile_config"`

	Address,
	InstanceID,
	ProxyJump,
	Port,
	User string

	// Names of the instance, meaning all aliases.
	// The main identifier is constructed from profile name and instance Name tag
	// then comes instance id, then there are a couple of more
	Names []string
}

// ConfigFormat returns formatted and stringified SSHEntry ready to use in ssh config
func (e SSHEntry) ConfigFormat() string {
	var output = []string{}

	output = append(output, fmt.Sprintf("Host %s", strings.Join(e.Names, " ")))

	if e.User != "" {
		output = append(output, fmt.Sprintf("    User %s", e.User))
	}
	if e.ProxyJump != "" {
		output = append(output, fmt.Sprintf("    ProxyJump %s", e.ProxyJump))
	}
	if e.Port != "" {
		output = append(output, fmt.Sprintf("    Port %s", e.Port))
	}
	output = append(output, fmt.Sprintf("    Hostname %s", e.Address), "\n")

	return strings.Join(output, "\n")
}
