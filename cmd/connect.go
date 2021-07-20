package cmd

import (
	"aws-ssh/lib"
	"aws-ssh/lib/cache"
	"aws-ssh/lib/ec2connect"
	"path"

	"github.com/apex/log"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var connectCmd = &cobra.Command{
	Use:   "connect [ssh command (ssh -tt user@instanceid)]",
	Short: "SSH into the EC2 instance using ec2 connect feature",
	// override the default prerun
	// as we don't need any profiles here
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
	},
	Long: `aws-ssh connects to the EC2 instance using ec2 connect feature. It makes a special API call to upload
the first public key from your running ssh agent and then runs ssh command`,
	Aliases: []string{"ssh"},
	/* There are 2 modes of this command:
	   1. Run with the specified instanceid and AWS profile
	   2. Get the above values from the cache.

	   Obviously, the second mode requires cache.
	   We just check if there are any parameters passed, and if not,
	   switch to the mode 2
	*/
	Run: func(cmd *cobra.Command, args []string) {
		var sshEntries lib.SSHEntries
		var profile string
		var instanceID = viper.GetString("instanceid")
		var defaultUser = viper.GetString("user")

		profiles := viper.GetStringSlice("profiles")
		if len(profiles) > 0 {
			profile = profiles[0]
			ec2connect.ConnectEC2(
				lib.SSHEntries{
					&lib.SSHEntry{
						ProfileConfig: lib.ProfileConfig{Name: profile},
						InstanceID:    instanceID,
						User:          defaultUser,
						Names:         []string{instanceID},
					},
				},
				viper.GetString("ssh-config-path"),
				args,
			)
		}
		// ok, profile is not set, switch to mode 2
		log.Info("No profile has been provided, switching to the cache mode")
		cache := cache.NewYAMLCache(viper.GetString("cache-dir"))

		sshEntry, err := cache.Lookup(instanceID)
		if err != nil {
			log.WithError(err).Fatalf("can't lookup %s in cache", instanceID)
		}
		if sshEntry.User == "" {
			sshEntry.User = defaultUser
		}

		sshEntries = append(sshEntries, &sshEntry)
		// ProxyJump is set, which means we need to lookup the bastion host too
		if sshEntry.ProxyJump != "" {
			bastionEntry, err := cache.Lookup(sshEntry.ProxyJump)
			if err != nil {
				log.WithError(err).Fatalf("can't lookup bastion %s in cache", sshEntry.ProxyJump)
			}
			if bastionEntry.User == "" {
				bastionEntry.User = defaultUser
			}
			log.WithField("instance_id", bastionEntry.InstanceID).Infof("Got bastion %s", bastionEntry.Names[0])
			sshEntries = append(sshEntries, &bastionEntry)
		}
		ec2connect.ConnectEC2(sshEntries, viper.GetString("ssh-config-path"), args)
	},
}

func init() {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.WithError(err).Fatal("can't get homedir")
	}
	defaultSSHConfigFile := path.Join(homeDir, ".ssh", "ec2_connect_config")

	connectCmd.Flags().StringP("instanceid", "i", "", "Instance ID to connect to")
	connectCmd.Flags().StringP("user", "u", "ec2-user", "Existing user on the instance")
	connectCmd.Flags().StringP("ssh-config-path", "c", defaultSSHConfigFile, "Existing user on the instance")
	viper.BindPFlag("instanceid", connectCmd.Flags().Lookup("instanceid"))
	viper.BindPFlag("user", connectCmd.Flags().Lookup("user"))
	viper.BindPFlag("ssh-config-path", connectCmd.Flags().Lookup("ssh-config-path"))
	rootCmd.AddCommand(connectCmd)
}
