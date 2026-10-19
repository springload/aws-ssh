package lib

import (
	"context"
	"fmt"
	"sort"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	multierror "github.com/hashicorp/go-multierror"
)

// profileSummary represents profile summary
// with raw unprocessed information about instances
type profileSummary struct {
	ProfileConfig

	Instances []types.Instance
}

// ProcessedProfileSummary represents profile summary
// with processed ssh entries, containing instance names, etc
type ProcessedProfileSummary struct {
	ProfileConfig

	SSHEntries []SSHEntry
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
	for _, summary := range profileSummaries {
		processedProfileSummaries = append(processedProfileSummaries, processProfileSummary(summary, noProfilePrefix))
	}
	return processedProfileSummaries, errors
}

// processProfileSummary turns raw instances of a profile into ssh entries
func processProfileSummary(summary profileSummary, noProfilePrefix bool) ProcessedProfileSummary {
	var profileSSHEntries []SSHEntry

	ctx := log.WithField("profile", summary.Name)
	// group instances by VPC
	ctx.Debug("Grouping instances by VPC")

	// sort by name first, then by launch time
	sorted := sortInstances(summary.Instances)

	vpcGroups := groupInstances(sorted, func(i types.Instance) string { return aws.ToString(i.VpcId) })

	commonBastions := filterInstances(sorted, func(i types.Instance) bool {
		return isBastionFromTags(i.Tags, true) // check for global tag as well
	})

	ctx.Debugf("Found %d common (global) bastions", len(commonBastions))

	for _, vpcGroup := range vpcGroups { // take the instances grouped by vpc and iterate
		vpcBastions := filterInstances(vpcGroup.Instances, func(i types.Instance) bool {
			return isBastionFromTags(i.Tags, false) // "false" means don't check for global tag
		})

		ctx.WithField("vpc", vpcGroup.Key).Debugf("Found %d bastions", len(vpcBastions))

		// now group them by name
		nameGroups := groupInstances(vpcGroup.Instances, func(i types.Instance) string { return getNameFromTags(i.Tags) })

		// now we have instances, grouped by vpc and name
		for _, nameGroup := range nameGroups {
			instanceName := nameGroup.Key

			for n, instance := range nameGroup.Instances {
				var entry = SSHEntry{
					InstanceID: aws.ToString(instance.InstanceId),
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
				entry.Address = aws.ToString(instance.PrivateIpAddress) // get the private address first as we always have one
				if bastion != nil {                                     // get private address and add proxyhost, which is the bastion ip
					// refer to the bastion by its instance ID
					// which we should have a record for
					entry.ProxyJump = aws.ToString(bastion.InstanceId)
				} else { // get public IP if we have one
					if publicIP := aws.ToString(instance.PublicIpAddress); publicIP != "" {
						entry.Address = aws.ToString(instance.PublicIpAddress)
					}
				}
				var instanceIndex string
				if len(nameGroup.Instances) > 1 {
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

	return ProcessedProfileSummary{
		ProfileConfig: ProfileConfig{
			Name:   summary.Name,
			Region: summary.Region,
			Domain: summary.Domain,
		},
		SSHEntries: profileSSHEntries,
	}
}

// DescribeProfile describes the specified profile
func DescribeProfile(profile ProfileConfig, sum chan profileSummary, errChan chan error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(profile.Name))

	if err != nil {
		errChan <- fmt.Errorf("couldn't create session for '%s': %s", profile.Name, err)
		return
	}

	profileSummary := profileSummary{
		ProfileConfig: ProfileConfig{
			Name:   profile.Name,
			Region: cfg.Region,
			Domain: profile.Domain,
		},
	}

	svc := ec2.NewFromConfig(cfg)
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{string(types.InstanceStateNameRunning)},
			},
		},
	}

	paginator := ec2.NewDescribeInstancesPaginator(svc, input)
	for paginator.HasMorePages() {
		result, subErr := paginator.NextPage(context.TODO())
		if subErr != nil {
			err = subErr
			break
		} else {
			for _, reservation := range result.Reservations {
				profileSummary.Instances = append(profileSummary.Instances, reservation.Instances...)
			}
		}
	}

	if err != nil {
		errChan <- fmt.Errorf("can't get full information for '%s': %s", profile, err)
	} else {
		sum <- profileSummary
	}
}
