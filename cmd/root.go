package cmd

import (
	"aws-ssh/lib"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	homedir "github.com/mitchellh/go-homedir"
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
	var defaultCacheDir string
	if val, ok := os.LookupEnv("XDG_CACHE_HOME"); ok {
		defaultCacheDir = path.Join(val, "aws-ssh")
	} else {
		homeDir, err := homedir.Dir()
		if err != nil {
			log.WithError(err).Fatal("can't get homedir")
		}
		defaultCacheDir = path.Join(homeDir, ".cache", "aws-ssh")
	}

	cobra.OnInitialize(initSettings)

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Show debug output")
	rootCmd.PersistentFlags().BoolP("no-profile-prefix", "n", false, "Do not prefix host names with profile name")
	rootCmd.PersistentFlags().StringSliceP("profile", "p", []string{}, "Profiles to query. Can be specified multiple times. If not specified, goes through all profiles in ~/.aws/config and ~/.aws/credentials")
	rootCmd.PersistentFlags().StringP("cache-dir", "", defaultCacheDir, "Cache dir, which is used by \"update\" and \"connect\" commands")

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("no-profile-prefix", rootCmd.PersistentFlags().Lookup("no-profile-prefix"))
	viper.BindPFlag("profiles", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("cache-dir", rootCmd.PersistentFlags().Lookup("cache-dir"))

	viper.SetEnvPrefix("aws_ssh") // will be uppercased

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

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
