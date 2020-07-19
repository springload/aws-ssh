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
	linq "gopkg.in/ahmetb/go-linq.v3"
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
	Name,
	ProxyJump,
	Port,
	User,
	Profile,
	Domain string
}

// ConfigFormat returns formatted and stringified SSHEntry ready to use in ssh config
func (e SSHEntry) ConfigFormat() string {
	var output = []string{}

	if e.Domain != "" {
		output = append(output, fmt.Sprintf("Host %s %s %s.%s %s.%s", e.Name, e.InstanceID, e.Address, e.Profile, e.Name, e.Domain))
	} else {
		output = append(output, fmt.Sprintf("Host %s %s %s.%s", e.Name, e.InstanceID, e.Address, e.Profile))
	}

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
	profileSummaries, err := TraverseProfiles(profiles)
	if err != nil {
		log.WithError(err).Error("got some errors")
		return
	}

	var sshEntries []SSHEntry

	for _, summary := range profileSummaries {
		ctx := log.WithField("profile", summary.Name)
		// group instances by VPC
		ctx.Debug("Grouping instances by VPC")

		var vpcInstances []linq.Group

		// take the instances slice
		linq.From(summary.Instances).OrderBy(instanceNameSorter). // sort by name first
										ThenBy(instanceLaunchTimeSorter).         // then by launch time
										GroupBy(func(i interface{}) interface{} { // and then group by vpc
				vpcID := i.(*ec2.Instance).VpcId
				return aws.StringValue(vpcID)
			}, func(i interface{}) interface{} {
				return i.(*ec2.Instance)
			}).ToSlice(&vpcInstances)

		var commonBastions []*ec2.Instance
		linq.From(summary.Instances).OrderBy(instanceNameSorter). // sort by name first
										ThenBy(instanceLaunchTimeSorter). // then by launch time
										Where(
				func(f interface{}) bool {
					return isBastionFromTags(f.(*ec2.Instance).Tags, true) // check for global tag as well
				},
			).ToSlice(&commonBastions)

		ctx.Debugf("Found %d common bastions", len(commonBastions))

		for _, vpcGroup := range vpcInstances { // take the instances grouped by vpc and iterate
			var vpcBastions []*ec2.Instance
			linq.From(vpcGroup.Group).Where(
				func(f interface{}) bool {
					return isBastionFromTags(f.(*ec2.Instance).Tags, false) // don't check for global tag
				},
			).ToSlice(&vpcBastions)

			ctx.Debugf("Found %d bastions", len(vpcBastions))

			var nameInstances []linq.Group
			linq.From(vpcGroup.Group).GroupBy(func(i interface{}) interface{} { // now group them by name
				instanceName := getNameFromTags(i.(*ec2.Instance).Tags)
				return instanceName
			}, func(i interface{}) interface{} {
				return i.(*ec2.Instance)
			}).ToSlice(&nameInstances)

			// now we have instances, grouped by vpc and name
			for _, nameGroup := range nameInstances {
				instanceName := nameGroup.Key.(string)

				for n, instance := range nameGroup.Group {
					instance := instance.(*ec2.Instance)
					var instanceIndex string
					if len(nameGroup.Group) > 1 {
						instanceIndex = fmt.Sprintf("%d", n+1)
					}
					var entry = SSHEntry{
						InstanceID: aws.StringValue(instance.InstanceId),
						Name:       getInstanceCanonicalName(summary.Name, instanceName, instanceIndex),
						Profile:    summary.Name,
						Domain:     summary.Domain,
					}
					if noProfilePrefix {
						entry.Name = getInstanceCanonicalName("", instanceName, instanceIndex)
					}

					// first try to find a bastion from this vpc
					bastion := findBestBastion(instanceName, vpcBastions)
					if bastion == nil { // then try common ones
						bastion = findBestBastion(instanceName, commonBastions)
					}
					entry.Address = aws.StringValue(instance.PrivateIpAddress) // get the private address first as we always have one
					if bastion != nil {                                        // get private address and add proxyhost, which is the bastion ip
						bastionUser := getTagValue("x-aws-ssh-user", bastion.Tags)
						bastionPort := getTagValue("x-aws-ssh-port", bastion.Tags)
						entry.ProxyJump = aws.StringValue(bastion.PublicIpAddress)
						if bastionUser != "" {
							entry.ProxyJump = fmt.Sprintf("%s@%s", bastionUser, entry.ProxyJump)
						}
						if bastionPort != "" {
							entry.ProxyJump = fmt.Sprintf("%s:%s", entry.ProxyJump, bastionPort)
						}
					} else { // get public IP if we have one
						if publicIP := aws.StringValue(instance.PublicIpAddress); publicIP != "" {
							entry.Address = aws.StringValue(instance.PublicIpAddress)
						}
					}
					entry.User = getTagValue("x-aws-ssh-user", instance.Tags)
					entry.Port = getTagValue("x-aws-ssh-port", instance.Tags)
					sshEntries = append(sshEntries, entry)
				}
			}
		}
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
