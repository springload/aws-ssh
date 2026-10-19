package lib

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func tag(k, v string) types.Tag { return types.Tag{Key: aws.String(k), Value: aws.String(v)} }

func inst(id, vpc, name, private, public string, launched time.Time, extra ...types.Tag) types.Instance {
	i := types.Instance{
		InstanceId:       aws.String(id),
		VpcId:            aws.String(vpc),
		PrivateIpAddress: aws.String(private),
		LaunchTime:       aws.Time(launched),
		Tags:             append([]types.Tag{tag("Name", name)}, extra...),
	}
	if public != "" {
		i.PublicIpAddress = aws.String(public)
	}
	return i
}

func testSummary() profileSummary {
	early := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	return profileSummary{
		ProfileConfig: ProfileConfig{Name: "prod", Region: "ap-southeast-2", Domain: "example.com"},
		// deliberately unordered
		Instances: []types.Instance{
			inst("i-web2", "vpc-a", "web", "10.0.0.2", "", late),
			inst("i-worker", "vpc-c", "worker", "10.2.0.1", "", early),
			inst("i-shared", "vpc-b", "shared-bastion", "10.1.0.9", "9.9.9.9", early, tag("Global", "yes")),
			inst("i-db", "vpc-a", "db", "10.0.0.3", "1.1.1.1", early, tag("x-aws-ssh-port", "2222")),
			inst("i-api", "vpc-b", "api", "10.1.0.1", "", early),
			inst("i-web1", "vpc-a", "web", "10.0.0.1", "", early, tag("x-aws-ssh-user", "ec2-user")),
			inst("i-bast-a", "vpc-a", "prod-bastion", "10.0.0.9", "8.8.8.8", early),
		},
	}
}

func entry(id, addr, jump, port, user string, names ...string) SSHEntry {
	return SSHEntry{
		ProfileConfig: ProfileConfig{Name: "prod", Region: "ap-southeast-2", Domain: "example.com"},
		InstanceID:    id,
		Address:       addr,
		ProxyJump:     jump,
		Port:          port,
		User:          user,
		Names:         names,
	}
}

func TestProcessProfileSummary(t *testing.T) {
	tests := []struct {
		name            string
		noProfilePrefix bool
		want            []SSHEntry
	}{
		{
			name: "with profile prefix",
			want: []SSHEntry{
				entry("i-api", "10.1.0.1", "i-shared", "", "", "prod-api", "i-api", "10.1.0.1.prod", "prod-api.example.com"),
				entry("i-bast-a", "8.8.8.8", "", "", "", "prod-bastion", "i-bast-a", "8.8.8.8.prod", "prod-bastion.example.com"),
				entry("i-db", "10.0.0.3", "i-bast-a", "2222", "", "prod-db", "i-db", "10.0.0.3.prod", "prod-db.example.com"),
				entry("i-shared", "9.9.9.9", "", "", "", "prod-shared-bastion", "i-shared", "9.9.9.9.prod", "prod-shared-bastion.example.com"),
				entry("i-web1", "10.0.0.1", "i-bast-a", "", "ec2-user", "prod-web-1", "i-web1", "10.0.0.1.prod", "prod-web-1.example.com"),
				entry("i-web2", "10.0.0.2", "i-bast-a", "", "", "prod-web-2", "i-web2", "10.0.0.2.prod", "prod-web-2.example.com"),
				entry("i-worker", "10.2.0.1", "i-shared", "", "", "prod-worker", "i-worker", "10.2.0.1.prod", "prod-worker.example.com"),
			},
		},
		{
			name:            "without profile prefix",
			noProfilePrefix: true,
			want: []SSHEntry{
				entry("i-api", "10.1.0.1", "i-shared", "", "", "api", "i-api", "10.1.0.1.prod", "api.example.com"),
				entry("i-db", "10.0.0.3", "i-bast-a", "2222", "", "db", "i-db", "10.0.0.3.prod", "db.example.com"),
				entry("i-bast-a", "8.8.8.8", "", "", "", "prod-bastion", "i-bast-a", "8.8.8.8.prod", "prod-bastion.example.com"),
				entry("i-shared", "9.9.9.9", "", "", "", "shared-bastion", "i-shared", "9.9.9.9.prod", "shared-bastion.example.com"),
				entry("i-web1", "10.0.0.1", "i-bast-a", "", "ec2-user", "web-1", "i-web1", "10.0.0.1.prod", "web-1.example.com"),
				entry("i-web2", "10.0.0.2", "i-bast-a", "", "", "web-2", "i-web2", "10.0.0.2.prod", "web-2.example.com"),
				entry("i-worker", "10.2.0.1", "i-shared", "", "", "worker", "i-worker", "10.2.0.1.prod", "worker.example.com"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// run repeatedly, group order must not leak into the result
			for n := 0; n < 20; n++ {
				got := processProfileSummary(testSummary(), tt.noProfilePrefix)
				if !reflect.DeepEqual(got.SSHEntries, tt.want) {
					t.Fatalf("run %d:\n got: %+v\nwant: %+v", n, got.SSHEntries, tt.want)
				}
			}
		})
	}
}
