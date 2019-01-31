### What it is
This program goes through all available AWS accounts in parallel and determines

IP addresses of ec2 instances. It also detects so-called "bastion" instances.

If a bastion instance has tag "Global" with value "yes", "true" or "1", then aws-ssh decides it can be
used across multiple VPCs. If there are multiple bastion instances, it chooses the instance that has the most common match in name.

Any comments and especially pull requests are highly appreciated.

```
Usage:
  aws-ssh [command]

Available Commands:
  help        Help about any command
  reconf      Creates a new ssh config
  test        A brief description of your command

Flags:
  -d, --debug             Show debug output
  -h, --help              help for aws-ssh
  -p, --profile strings   Profiles to query. Can be specified multiple times. If not specified, goes through all profiles in ~/.aws/config and ~/.aws/credentials
      --version           version for aws-ssh

Use "aws-ssh [command] --help" for more information about a command.
```

### Build

You'll need go>=1.11. Note that this project uses `go.mod`, so the project has to be cloned somewhere outside of the `GOPATH` directory.
Or just use provided `Dockerfile`.
