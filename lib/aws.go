package lib

import (
	"fmt"
	"sort"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	multierror "github.com/hashicorp/go-multierror"
	linq "gopkg.in/ahmetb/go-linq.v3"
)

// profileSummary represents profile summary
// with raw unprocessed information about instances
type profileSummary struct {
	ProfileConfig

	Instances []*ec2.Instance
}

// ProcessedProfileSummary represents profile summary
// with processed ssh entries, containing instance names, etc
type ProcessedProfileSummary struct {
	ProfileConfig

	SSHEntries []SSHEntry
}

func makeSession(profile string) (*session.Session, error) {
	log.Debugf("Creating session for %s", profile)
	// create AWS session
	localSession, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{},

		SharedConfigState: session.SharedConfigEnable,
		Profile:           profile,
	})
	if err != nil {
		return nil, fmt.Errorf("can't get aws session")
	}
	return localSession, nil
}

// TraverseProfiles goes through all profiles and returns a list of ProcessedProfileSummary
func TraverseProfiles(profiles []ProfileConfig, noProfilePrefix bool) ([]ProcessedProfileSummary, error) {
	log.Debugf("Traversing through %d profiles", len(profiles))
	var profileSummaryChan = make(chan profileSummary, len(profiles))
	var errChan = make(chan error, len(profiles))

	var profileSummaries []profileSummary
	for _, profile := range profiles {
		go func(profile ProfileConfig) {
			DescribeProfile(profile, profileSummaryChan, errChan)
		}(profile)
	}

	var errors error // errors collector

	for n := 0; n < len(profiles); n++ {
		select {
		case summary := <-profileSummaryChan:
			profileSummaries = append(profileSummaries, summary)
		case err := <-errChan:
			errors = multierror.Append(errors, err)
		}
	}

	// sort alphabetically by profile name
	sort.Slice(profileSummaries, func(i, j int) bool { return profileSummaries[i].Name < profileSummaries[j].Name })

	var processedProfileSummaries []ProcessedProfileSummary
	// go through all profileSummaries and
	// create sshEntries out of it
	for _, summary := range profileSummaries {
		var profileSSHEntries []SSHEntry

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

		ctx.Debugf("Found %d common (global) bastions", len(commonBastions))

		for _, vpcGroup := range vpcInstances { // take the instances grouped by vpc and iterate
			var vpcBastions []*ec2.Instance
			linq.From(vpcGroup.Group).Where(
				func(f interface{}) bool {
					return isBastionFromTags(f.(*ec2.Instance).Tags, false) // "false" means don't check for global tag
				},
			).ToSlice(&vpcBastions)

			ctx.WithField("vpc", vpcGroup.Key).Debugf("Found %d bastions", len(vpcBastions))

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
					var entry = SSHEntry{
						InstanceID: aws.StringValue(instance.InstanceId),
						ProfileConfig: ProfileConfig{
							Name:   summary.Name,
							Region: summary.Region,
							Domain: summary.Domain,
						},
					}
					entry.User = GetUserFromTags(instance.Tags)
					entry.Port = getPortFromTags(instance.Tags)

					// first try to find a bastion from this vpc
					bastion := findBestBastion(instanceName, vpcBastions)
					if bastion == nil { // then try common ones
						bastion = findBestBastion(instanceName, commonBastions)
					}
					entry.Address = aws.StringValue(instance.PrivateIpAddress) // get the private address first as we always have one
					if bastion != nil {                                        // get private address and add proxyhost, which is the bastion ip
						// refer to the bastion by its instance ID
						// which we should have a record for
						entry.ProxyJump = aws.StringValue(bastion.InstanceId)
					} else { // get public IP if we have one
						if publicIP := aws.StringValue(instance.PublicIpAddress); publicIP != "" {
							entry.Address = aws.StringValue(instance.PublicIpAddress)
						}
					}
					var instanceIndex string
					if len(nameGroup.Group) > 1 {
						instanceIndex = fmt.Sprintf("%d", n+1)
					}
					// add all names of the instance
					var name = getInstanceCanonicalName(summary.Name, instanceName, instanceIndex)
					if noProfilePrefix {
						name = getInstanceCanonicalName("", instanceName, instanceIndex)
					}
					entry.Names = append(entry.Names, name, entry.InstanceID, fmt.Sprintf("%s.%s", entry.Address, entry.ProfileConfig.Name))
					if summary.Domain != "" {
						entry.Names = append(entry.Names, fmt.Sprintf("%s.%s", name, summary.Domain))
					}
					profileSSHEntries = append(profileSSHEntries, entry)
				}
			}
		}
		// sort by the first (main) name alphabetically
		sort.SliceStable(profileSSHEntries, func(i, j int) bool { return profileSSHEntries[i].Names[0] < profileSSHEntries[j].Names[0] })

		processedProfileSummaries = append(processedProfileSummaries, ProcessedProfileSummary{
			ProfileConfig: ProfileConfig{
				Name:   summary.Name,
				Region: summary.Region,
				Domain: summary.Domain,
			},
			SSHEntries: profileSSHEntries,
		})
	}
	return processedProfileSummaries, errors
}

// DescribeProfile describes the specified profile
func DescribeProfile(profile ProfileConfig, sum chan profileSummary, errChan chan error) {
	awsSession, err := makeSession(profile.Name)
	if err != nil {
		errChan <- fmt.Errorf("Couldn't create session for '%s': %s", profile.Name, err)
		return
	}

	profileSummary := profileSummary{
		ProfileConfig: ProfileConfig{
			Name:   profile.Name,
			Region: aws.StringValue(awsSession.Config.Region),
			Domain: profile.Domain,
		},
	}

	svc := ec2.New(awsSession)
	input := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: aws.StringSlice([]string{ec2.InstanceStateNameRunning}),
			},
		},
	}

	err = svc.DescribeInstancesPages(input, func(result *ec2.DescribeInstancesOutput, lastPage bool) bool {
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				profileSummary.Instances = append(profileSummary.Instances, instance)
			}
		}
		return false
	})
	if err != nil {
		errChan <- fmt.Errorf("Can't get full information for '%s': %s", profile, err)
	} else {
		sum <- profileSummary
	}
}
