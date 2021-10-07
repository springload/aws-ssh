package lib

import (
	"regexp"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const bastionCanonicalName = "bastion"

var sanitiser = regexp.MustCompile("[\\s-]+")

func getTagValue(tag string, tags []types.Tag, caseInsensitive ...bool) string {
	if len(caseInsensitive) > 0 {
		if caseInsensitive[0] {
			tag = strings.ToLower(tag)
		}
	}

	for _, subTag := range tags {
		if aws.ToString(subTag.Key) == tag {
			return aws.ToString(subTag.Value)
		}
	}

	return ""

}

func getNameFromTags(tags []types.Tag) string {
	return strings.ToLower(getTagValue("Name", tags))
}

func getPortFromTags(tags []types.Tag) string {
	return strings.ToLower(getTagValue("x-aws-ssh-port", tags))
}

// GetUserFromTags gets the ec2 username from tags
func GetUserFromTags(tags []types.Tag) string {
	return strings.ToLower(getTagValue("x-aws-ssh-user", tags))
}

func isBastionFromTags(tags []types.Tag, checkGlobal bool) bool {
	if len(tags) > 0 {
		var name string
		var global bool

		for _, tag := range tags {
			switch aws.ToString(tag.Key) {
			case "Name":
				name = strings.ToLower(aws.ToString(tag.Value))
			case "Global", "x-aws-ssh-global":
				{
					value := strings.ToLower(aws.ToString(tag.Value))
					if value == "yes" || value == "true" || value == "1" {
						global = true
					}
				}
			}
		}

		if strings.Contains(name, bastionCanonicalName) {
			if checkGlobal {
				if global {
					return true
				}
			} else {
				return true
			}
		}
	}
	return false
}

type weightType struct {
	Index, Weight int
}

type weights []weightType

func (w weights) Len() int           { return len(w) }
func (w weights) Less(i, j int) bool { return w[i].Weight < w[j].Weight }
func (w weights) Swap(i, j int)      { w[i], w[j] = w[j], w[i] }

func findBestBastion(instanceName string, bastions []*types.Instance) *types.Instance {
	// skip instances with bastionCanonicalName in name
	if !strings.Contains(instanceName, bastionCanonicalName) && len(bastions) > 0 {
		if len(bastions) == 1 {
			return bastions[0]
		}

		var weights weights
		for n, bastion := range bastions {
			bastionName := getNameFromTags(bastion.Tags)
			weight := len(lcs(instanceName, bastionName))
			weights = append(weights, weightType{Index: n, Weight: weight})
		}
		// sort by weiht
		sort.Sort(weights)
		// return the first one
		return bastions[weights[0].Index]
	}

	return nil
}

func getInstanceCanonicalName(profile, instanceName, instanceIndex string) string {
	var parts []string
	if !strings.HasPrefix(instanceName, profile) {
		parts = append(parts, profile)
	}
	parts = append(parts, instanceName)

	if instanceIndex != "" {
		parts = append(parts, instanceIndex)
	}

	return sanitiser.ReplaceAllString(strings.Join(parts, "-"), "-")
}

func instanceLaunchTimeSorter(i interface{}) interface{} { // sorts by launch time
	launched := aws.ToTime(i.(*types.Instance).LaunchTime)
	return launched.Unix()
}

func instanceNameSorter(i interface{}) interface{} { // sort by instance name
	instanceName := getNameFromTags(i.(*types.Instance).Tags)
	return instanceName
}
