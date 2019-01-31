package cmd

import (
	"aws-ssh/lib"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		summaries, err := lib.TraverseProfiles(viper.GetStringSlice("profiles"))
		if err != nil {
			log.WithError(err).Fatal("Can't traverse through all profiles")
		} else {
			log.Info("All profiles have been traversted through without errors")
			for _, summary := range summaries {
				log.WithFields(log.Fields{"profile": summary.Name}).Infof("found %d instances", len(summary.Instances))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
