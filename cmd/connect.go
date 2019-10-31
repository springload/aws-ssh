package cmd

import (
	"aws-ssh/lib"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var connectCmd = &cobra.Command{
	Use:   "connect [ssh command (ssh -tt user@instanceid)]",
	Short: "SSH into the EC2 instance using ec2 connect feature",
	Long: `aws-ssh connects to the EC2 instance using ec2 connect feature. It makes a special API call to upload
the first public key from your running ssh agent and then runs ssh command`,
	Aliases: []string{"ssh"},
	Run: func(cmd *cobra.Command, args []string) {
		var profile string

		profiles := viper.GetStringSlice("profiles")
		if len(profiles) > 0 {
			profile = profiles[0]
		}
		lib.ConnectEC2(profile, viper.GetString("instanceid"), viper.GetString("user"), args)
	},
}

func init() {
	connectCmd.Flags().StringP("instanceid", "i", "", "Instance ID to connect to")
	connectCmd.Flags().StringP("user", "u", "ec2-user", "Existing user on the instance")
	connectCmd.MarkFlagRequired("instanceid")

	viper.BindPFlag("instanceid", connectCmd.Flags().Lookup("instanceid"))
	viper.BindPFlag("user", connectCmd.Flags().Lookup("user"))
	rootCmd.AddCommand(connectCmd)
}
