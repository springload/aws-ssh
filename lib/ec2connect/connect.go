package ec2connect

import (
	"aws-ssh/lib"

	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2instanceconnect"
	"golang.org/x/crypto/ssh/agent"
)

// ConnectEC2 connects to an EC2 instance by pushing your public key onto it first
// using EC2 connect feature and then runs ssh.
func ConnectEC2(sshEntries lib.SSHEntries, sshConfigPath string, args []string) {
	// get the pub key from the ssh agent first
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))

	keys, err := agent.NewClient(sshAgent).List()
	if err != nil || len(keys) < 1 {
		log.Fatal("Can't get public keys from ssh agent. Please ensure you have the ssh-agent running and have at least one identity added (with ssh-add)")
	}
	pubkey := keys[0].String()

	// push the pub key to those instances one after each other
	// TODO: make it parallel
	for _, sshEntry := range sshEntries {
		var instanceName = sshEntry.InstanceID
		if len(sshEntry.Names) > 0 {
			instanceName = sshEntry.Names[0]
		}

		log.WithField("instance", instanceName).WithField("user", sshEntry.User).Info("Pushing SSH key...")
		instanceIpAddress, err := pushEC2Connect(sshEntry.ProfileConfig.Name, sshEntry.InstanceID, sshEntry.User, pubkey)
		if err != nil {
			log.WithError(err).Fatal("can't push ssh key to the instance")
		}
		// if the address is empty we set to the value we got from ec2 connect push
		if sshEntry.Address == "" {
			sshEntry.Address = instanceIpAddress
		}
	}

	// then generate ssh config for all instances in sshEntries
	// save the dynamic ssh config first
	if err := sshEntries.SaveConfig(sshConfigPath); err != nil {
		log.WithError(err).Fatal("can't save ssh config for ec2 connect")
	}

	var instanceName = sshEntries[0].InstanceID
	if len(sshEntries[0].Names) > 0 {
		instanceName = sshEntries[0].Names[0]
	}
	// connect to the first instance in sshEntry, as the other will be bastion(s)
	if len(args) == 0 {
		// construct default args
		args = []string{
			"ssh",
			"-tt",
			instanceName,
		}
	}

	command, err := exec.LookPath(args[0])
	if err != nil {
		log.WithError(err).Fatal("Can't find the binary in the PATH")
	}
	log.WithField("instance_id", sshEntries[0].InstanceID).Infof("Connecting to the instance using '%s'", strings.Join(args, " "))

	if err := syscall.Exec(command, args, os.Environ()); err != nil {
		log.WithFields(log.Fields{"command": command}).WithError(err).Fatal("can't run the command")
	}
}

// pushEC2Connect pushes the ssh key to a given profile and instance ID
// and returns the public (or private if public doesn't exist) address of the EC2 instance
func pushEC2Connect(profile, instanceID, instanceUser, pubKey string) (string, error) {
	localSession, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{},

		SharedConfigState: session.SharedConfigEnable,
		Profile:           profile,
	})
	if err != nil {
		return "", fmt.Errorf("can't get aws session: %s", err)
	}
	ec2Svc := ec2.New(localSession)
	ec2Result, err := ec2Svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	})
	if err != nil {
		return "", fmt.Errorf("can't get ec2 instance: %s", err)
	}

	if len(ec2Result.Reservations) == 0 || len(ec2Result.Reservations[0].Instances) == 0 {
		return "", fmt.Errorf("Couldn't find the instance %s", instanceID)
	}

	ec2Instance := ec2Result.Reservations[0].Instances[0]
	ec2ICSvc := ec2instanceconnect.New(localSession)

	if _, err := ec2ICSvc.SendSSHPublicKey(&ec2instanceconnect.SendSSHPublicKeyInput{
		InstanceId:       ec2Instance.InstanceId,
		InstanceOSUser:   aws.String(instanceUser),
		AvailabilityZone: ec2Instance.Placement.AvailabilityZone,
		SSHPublicKey:     aws.String(pubKey),
	}); err != nil {
		return "", fmt.Errorf("can't push ssh key: %s", err)
	}
	var address = aws.StringValue(ec2Instance.PrivateIpAddress)
	if aws.StringValue(ec2Instance.PublicIpAddress) != "" {
		address = aws.StringValue(ec2Instance.PublicIpAddress)
	}
	return address, nil
}
