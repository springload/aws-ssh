package lib

// ProfileConfig represents an entry in aws config
type ProfileConfig struct {
	Name, // aws profile name
	Region, // region
	Domain string // domain if set with "aws-ssh-domain" in the config
}
