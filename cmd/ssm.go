package cmd

import (
	"aws-ssh/lib"
    "aws-ssh/lib/cache"
    "aws-ssh/lib/ec2connect"
    "github.com/apex/log"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var ssmCmd = &cobra.Command{
    Use:   "ssm connect",
    Short: "Connect into the EC2 instance using SSM",
    Long: `This command initiates an SSM session to an EC2 instance, using instance ID,
or retrieving the information from a cache if no parameters are explicitly provided.`,
    Run: func(cmd *cobra.Command, args []string) {
		var sshEntries lib.SSHEntries
		var instanceID = viper.GetString("instanceid")
		cache := cache.NewYAMLCache(viper.GetString("cache-dir"))

			sshEntry, err := cache.Lookup(instanceID)
			if err != nil {
				log.WithError(err).Fatalf("can't lookup %s in cache", instanceID)
			}
	
			sshEntries = append(sshEntries, &sshEntry)
			ec2connect.SsmEC2(sshEntries, args)
		
	},
}

func init() {

    ssmCmd.Flags().StringP("instanceid", "i", "", "Instance ID to connect to")
    viper.BindPFlags(ssmCmd.Flags())
    viper.AutomaticEnv()
    rootCmd.AddCommand(ssmCmd)
}
