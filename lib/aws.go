package lib

import (
	"fmt"
	"sort"
	"sync"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	multierror "github.com/hashicorp/go-multierror"
)

// ProfileSummary represents profile summary
type ProfileSummary struct {
	sync.Mutex

	Name      string
	Region    string
	Instances []*ec2.Instance
	Domain    string
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

// TraverseProfiles goes through all profiles and returns a list of ProfileSummary
func TraverseProfiles(profiles []ProfileConfig) ([]ProfileSummary, error) {
	log.Debugf("Traversing through %d profiles", len(profiles))
	var profileSummaryChan = make(chan ProfileSummary, len(profiles))
	var errChan = make(chan error, len(profiles))

	var profileSummaries []ProfileSummary
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
	return profileSummaries, errors
}

// DescribeProfile describes the specified profile
func DescribeProfile(profile ProfileConfig, sum chan ProfileSummary, errChan chan error) {
	awsSession, err := makeSession(profile.Name)
	if err != nil {
		errChan <- fmt.Errorf("Couldn't create session for '%s': %s", profile.Name, err)
		return
	}

	profileSummary := ProfileSummary{
		Name:   profile.Name,
		Region: aws.StringValue(awsSession.Config.Region),
		Domain: profile.Domain,
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
