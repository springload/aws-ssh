package cmd

import (
	"aws-ssh/lib"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var reconfCmd = &cobra.Command{
	Use: "reconf <filename>",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfig()
	},
	Args:  cobra.ExactArgs(1),
	Short: "Creates a new ssh config",
	Long: `Reconfigures your ssh by creating a new config for it. Only one argument is required,
which is a filename. In case of any errors, the preexisting file won't be touched.`,
	Run: func(cmd *cobra.Command, args []string) {
		lib.Reconf(viper.Get("profilesConfig").([]lib.ProfileConfig), args[0], viper.GetBool("no-profile-prefix"))
	},
}

func init() {
	rootCmd.AddCommand(reconfCmd)
}
