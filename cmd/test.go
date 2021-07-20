package cmd

import (
	"aws-ssh/lib"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Tests the profiles",
	Long: `aws-ssh test tests the AWS profiles found in
~/.aws/config and ~/.aws/credentials (unless -p option is provided)

Allows to identify permission issues early.
`,
	Run: func(cmd *cobra.Command, args []string) {
		profiles := viper.Get("profilesConfig").([]lib.ProfileConfig)
		summaries, err := lib.TraverseProfiles(profiles, viper.GetBool("no-profile-prefix"))
		if err != nil {
			log.WithError(err).Fatal("Can't traverse through all profiles")
		} else {
			log.Info("All profiles have been traversted through without errors")
			for _, summary := range summaries {
				log.WithFields(log.Fields{"profile": summary.Name}).Infof("found %d instances", len(summary.SSHEntries))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
