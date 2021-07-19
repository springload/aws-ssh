package lib

import (
	"testing"
)

var testdata = []struct {
	entry SSHEntry
	formatted,
	description string
}{
	{
		entry: SSHEntry{
			Address: "54.54.54.54",
			Names:   []string{"i-123456789", "some_custom_name"},
			User:    "ec2-user",
		},
		formatted: `Host i-123456789 some_custom_name
    User ec2-user
    Hostname 54.54.54.54

`, description: "basic entry"},
	{
		entry: SSHEntry{
			Address:   "54.54.54.54",
			Names:     []string{"i-123456789", "some_custom_name"},
			User:      "ubuntu",
			Port:      "2222",
			ProxyJump: "jumphost",
		},
		formatted: `Host i-123456789 some_custom_name
    User ubuntu
    ProxyJump jumphost
    Port 2222
    Hostname 54.54.54.54

`, description: "entry with jumphost and custom port"},
}

// TestConfigFormat tests ConfigFormat function of SSHEntry
// to make sure we get what we expect to get
func TestConfigFormat(t *testing.T) {
	for _, data := range testdata {
		var formatted = data.entry.ConfigFormat()

		if formatted == data.formatted {
			t.Logf("ConfigFormat() for \"%s\" matches what we expect", data.description)
		} else {
			t.Fatalf("%s\n%#v\n!=\n%#v", data.description, formatted, data.formatted)
		}
	}
}
