package cmd

import (
	"aws-ssh/lib"
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "aws-ssh",
	Short: "Describe your AWS and get ssh config to connect to ec2 instances",
	Long: `This program goes through all available AWS accounts in parallel and determines
IP addresses of ec2 instances. It also detects so-called "bastion" instances.

If a bastion instance has tag "Global" with value "yes", "true" or "1", then aws-ssh decides it can be
used across multiple VPCs. If there are multiple bastion instances, it chooses the instance that has the most common match in name.

Any comments and especially pull requests are highly appreciated.
`,
}

// Execute is a wrapper for rootCmd.Execute to add the version at compilation time
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initSettings)

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Show debug output")
	rootCmd.PersistentFlags().StringSliceP("profile", "p", []string{}, "Profiles to query. Can be specified multiple times. If not specified, goes through all profiles in ~/.aws/config and ~/.aws/credentials")

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("profiles", rootCmd.PersistentFlags().Lookup("profile"))
}

func initSettings() {
	log.SetHandler(cli.New(os.Stdout))
	if viper.GetBool("debug") {
		log.SetLevel(log.DebugLevel)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	profiles, err := getProfiles()
	if err != nil {
		log.WithError(err).Fatal("Profiles have not been provided and couldn't retrieve them from the config")
	}
	if len(viper.GetStringSlice("profiles")) == 0 {
		viper.Set("profilesConfig", profiles)
	} else {
		specifiedProfiles := viper.GetStringSlice("profiles")
		filteredProfiles := make([]lib.ProfileConfig, 0, len(specifiedProfiles))
		for _, profile := range profiles {
			if contains(specifiedProfiles, profile.Name) {
				filteredProfiles = append(filteredProfiles, profile)
			}
		}
		viper.Set("profilesConfig", filteredProfiles)
	}
}
