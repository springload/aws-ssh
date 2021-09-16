package cmd

import (
	"aws-ssh/lib"
	"aws-ssh/lib/cache"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates cache of ssh entries",
	Long: `
Cache is important for sophisticated behaviour of "connect" command,
because it stores metadata like AWS profile and EC2 instance id in it,
allowing to automatically fill those values in.
`,
	Run: func(cmd *cobra.Command, args []string) {
		cache := cache.NewYAMLCache(viper.GetString("cache-dir"))
		profileSummaries, err := lib.TraverseProfiles(viper.Get("profilesConfig").([]lib.ProfileConfig), viper.GetBool("no-profile-prefix"))
		if err != nil {
			log.WithError(err).Warn("got some errors")
		}

		var sshEntries []lib.SSHEntry

		// go through all profileSummaries and
		// create sshEntries out of it
		for _, summary := range profileSummaries {
			sshEntries = append(sshEntries, summary.SSHEntries...)
		}
		if err := cache.Save(profileSummaries); err != nil {
			log.WithError(err).Fatal("couldn't save cache")

		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// updateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// updateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
