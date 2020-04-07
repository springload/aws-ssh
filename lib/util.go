package lib

import (
	"regexp"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const bastionCanonicalName = "bastion"

var sanitiser = regexp.MustCompile("[\\s-]+")

func getTagValue(tag string, tags []*ec2.Tag, caseInsensitive ...bool) string {
	if len(caseInsensitive) > 0 {
		if caseInsensitive[0] {
			tag = strings.ToLower(tag)
		}
	}

	for _, subTag := range tags {
		if aws.StringValue(subTag.Key) == tag {
			return aws.StringValue(subTag.Value)
		}
	}

	return ""

}
func getNameFromTags(tags []*ec2.Tag) string {
	return strings.ToLower(getTagValue("Name", tags))
}

func isBastionFromTags(tags []*ec2.Tag, checkGlobal bool) bool {
	if len(tags) > 0 {
		var name string
		var global bool

		for _, tag := range tags {
			switch aws.StringValue(tag.Key) {
			case "Name":
				name = strings.ToLower(aws.StringValue(tag.Value))
			case "Global", "x-aws-ssh-global":
				{
					value := strings.ToLower(aws.StringValue(tag.Value))
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

func findBestBastion(instanceName string, bastions []*ec2.Instance) *ec2.Instance {
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
	parts = append(parts, instanceName, instanceIndex)

	return sanitiser.ReplaceAllString(strings.Join(parts, "-"), "-")
}
