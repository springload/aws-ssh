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

func getNameFromTags(tags []*ec2.Tag) string {
	if len(tags) > 0 {
		for _, tag := range tags {
			if aws.StringValue(tag.Key) == "Name" {
				return strings.ToLower(aws.StringValue(tag.Value))
			}
		}
	}

	return ""
}

func isBastionFromTags(tags []*ec2.Tag, checkGlobal bool) bool {
	if len(tags) > 0 {
		var name string
		var global bool

		for _, tag := range tags {
			switch aws.StringValue(tag.Key) {
			case "Name":
				name = strings.ToLower(aws.StringValue(tag.Value))
			case "Global":
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
		} else {
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
