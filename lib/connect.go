package lib

import (
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
func ConnectEC2(profile, instanceID, instanceUser string, args []string) {
	localSession, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{},

		SharedConfigState: session.SharedConfigEnable,
		Profile:           profile,
	})
	if err != nil {
		log.WithError(err).Fatal("can't get aws session")
	}
	ec2Svc := ec2.New(localSession)
	ec2Result, err := ec2Svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	})
	if err != nil {
		log.WithError(err).Fatal("can't get ec2 instance")
	}

	ec2Instance := ec2Result.Reservations[0].Instances[0]
	ec2ICSvc := ec2instanceconnect.New(localSession)

	log.WithField("instance_id", aws.StringValue(ec2Instance.InstanceId)).Info("Pushing SSH key...")

	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))

	keys, err := agent.NewClient(sshAgent).List()
	if err != nil || len(keys) < 1 {
		log.Fatal("Can't get public keys from ssh agent. Please ensure you have the ssh-agent running and have at least one identity added (with ssh-add)")
	}
	pubkey := keys[0].String()

	if _, err := ec2ICSvc.SendSSHPublicKey(&ec2instanceconnect.SendSSHPublicKeyInput{
		InstanceId:       ec2Instance.InstanceId,
		InstanceOSUser:   aws.String(instanceUser),
		AvailabilityZone: ec2Instance.Placement.AvailabilityZone,
		SSHPublicKey:     aws.String(pubkey),
	}); err != nil {
		log.WithError(err).Fatal("can't push ssh key")
	}

	if len(args) == 0 {
		// construct default args
		args = []string{
			"ssh",
			"-tt",
			fmt.Sprintf("%s@%s", instanceUser, instanceID),
		}
	}

	command, err := exec.LookPath(args[0])
	if err != nil {
		log.WithError(err).Fatal("Can't find the binary in the PATH")
	}
	log.WithField("instance_id", aws.StringValue(ec2Instance.InstanceId)).Infof("Connecting to the instance using '%s'", strings.Join(args, " "))

	if err := syscall.Exec(command, args, os.Environ()); err != nil {
		log.WithFields(log.Fields{"command": command}).WithError(err).Fatal("can't run the command")
	}
}
