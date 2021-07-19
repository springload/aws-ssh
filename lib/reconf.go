package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// ProfileConfig represents an entry in aws config
type ProfileConfig struct {
	Name,
	Domain string
}

// SSHEntry represents an entry in ssh config
type SSHEntry struct {
	Address,
	InstanceID,
	ProxyJump,
	Port,
	User,
	Profile string

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

func instanceLaunchTimeSorter(i interface{}) interface{} { // sorts by launch time
	launched := aws.TimeValue(i.(*ec2.Instance).LaunchTime)
	return launched.Unix()
}

func instanceNameSorter(i interface{}) interface{} { // sort by instance name
	instanceName := getNameFromTags(i.(*ec2.Instance).Tags)
	return instanceName
}

// Reconf writes ssh config with profiles into the specified file
func Reconf(profiles []ProfileConfig, filename string, noProfilePrefix bool) {
	profileSummaries, err := TraverseProfiles(profiles, noProfilePrefix)
	if err != nil {
		log.WithError(err).Error("got some errors")
		return
	}

	var sshEntries []SSHEntry

	// go through all profileSummaries and
	// create sshEntries out of it
	for _, summary := range profileSummaries {
		sshEntries = append(sshEntries, summary.SSHEntries...)
	}

	tmpfile, err := ioutil.TempFile(path.Dir(filename), "aws-ssh")
	ctx := log.WithField("tmpfile", tmpfile.Name())
	if err != nil {
		ctx.WithError(err).Fatal("Couldn't create a temporary file")
	}

	for _, entry := range sshEntries {
		if _, err := io.WriteString(tmpfile, entry.ConfigFormat()); err != nil {
			ctx.WithError(err).Fatal("Can't write to the temp file")
		}
	}
	if err := tmpfile.Close(); err != nil {
		ctx.WithError(err).Fatal("Couldn't close the temporary file")
	}
	if err := os.Rename(tmpfile.Name(), filename); err != nil {
		ctx.WithError(err).Fatalf("Couldn't move the file %s to %s", tmpfile.Name(), filename)
	}
}
