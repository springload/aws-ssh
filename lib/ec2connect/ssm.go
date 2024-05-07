package ec2connect

import (
    "aws-ssh/lib"
    "fmt"
    "os"
    "os/exec"
    "github.com/apex/log"
)

func SsmEC2(sshEntries lib.SSHEntries, args []string) {
    for _, sshEntry := range sshEntries {
        // Determine instance name for logging purposes
        instanceName := sshEntry.InstanceID
        if len(sshEntry.Names) > 0 {
            instanceName = sshEntry.Names[0]
        }
        log.WithField("instance", instanceName).Info("Trying to establish EC2 SSM session...")

        // Create command to start AWS SSM session via CLI
        cmd := exec.Command("aws", "ssm", "start-session", "--target", sshEntry.InstanceID, "--profile", sshEntry.ProfileConfig.Name)
        fmt.Println("Executing command:", cmd.String())

        // Redirect standard input/output streams
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr

        // Run the command and handle errors
        if err := cmd.Run(); err != nil {
            log.WithError(err).Errorf("Failed to start SSM session for instance %s", instanceName)
            break // Stop processing further instances on error
        }
    }
}
