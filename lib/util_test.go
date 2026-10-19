package lib

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func ids(instances []types.Instance) []string {
	var out []string
	for _, i := range instances {
		out = append(out, aws.ToString(i.InstanceId))
	}
	return out
}

func TestSortInstances(t *testing.T) {
	early := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	input := []types.Instance{
		inst("i-b-late", "v", "b", "1", "", late),
		inst("i-a-tie1", "v", "a", "1", "", early),
		inst("i-b-early", "v", "B", "1", "", early),
		inst("i-a-tie2", "v", "a", "1", "", early),
	}

	got := ids(sortInstances(input))
	want := []string{"i-a-tie1", "i-a-tie2", "i-b-early", "i-b-late"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if first := aws.ToString(input[0].InstanceId); first != "i-b-late" {
		t.Errorf("input was modified, first is now %s", first)
	}
}

func TestGroupInstances(t *testing.T) {
	early := time.Time{}
	input := []types.Instance{
		inst("i-1", "vpc-b", "x", "1", "", early),
		inst("i-2", "vpc-a", "x", "1", "", early),
		inst("i-3", "vpc-b", "x", "1", "", early),
	}

	groups := groupInstances(input, func(i types.Instance) string { return aws.ToString(i.VpcId) })

	var keys []string
	for _, g := range groups {
		keys = append(keys, g.Key)
	}
	if want := []string{"vpc-b", "vpc-a"}; !reflect.DeepEqual(keys, want) {
		t.Errorf("group keys: got %v, want %v (first-seen order)", keys, want)
	}
	if got, want := ids(groups[0].Instances), []string{"i-1", "i-3"}; !reflect.DeepEqual(got, want) {
		t.Errorf("first group: got %v, want %v", got, want)
	}
}

func TestFilterInstances(t *testing.T) {
	input := []types.Instance{
		inst("i-1", "v", "keep", "1", "", time.Time{}),
		inst("i-2", "v", "drop", "1", "", time.Time{}),
		inst("i-3", "v", "keep", "1", "", time.Time{}),
	}

	got := ids(filterInstances(input, func(i types.Instance) bool { return getNameFromTags(i.Tags) == "keep" }))
	if want := []string{"i-1", "i-3"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := filterInstances(input, func(types.Instance) bool { return false }); len(got) != 0 {
		t.Errorf("expected nothing, got %v", ids(got))
	}
}
