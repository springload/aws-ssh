### What it is

This program goes through all available AWS accounts in parallel and determines

IP addresses of ec2 instances. It also detects so-called "bastion" instances.

There are the following EC2 instance tags that change behaviour:

1. (Deprecated) If a bastion instance has tag "Global" with value "yes", "true" or "1", then aws-ssh will use it for all VPCs. If there are multiple bastion instances, it chooses the instance that has the most common match in name.
2. "x-aws-ssh-global" - same as the above
3. "x-aws-ssh-user" - sets the ssh username in the config.
4. "x-aws-ssh-port" - sets the ssh port in the config.

Any comments and especially pull requests are highly appreciated.

```
Usage:
  aws-ssh [command]

Available Commands:
  connect     SSH into the EC2 instance using ec2 connect feature
  help        Help about any command
  reconf      Creates a new ssh config
  test        A brief description of your command

Flags:
  -d, --debug               Show debug output
  -h, --help                help for aws-ssh
  -n, --no-profile-prefix   Do not prefix host names with profile name
  -p, --profile strings     Profiles to query. Can be specified multiple times. If not specified, goes through all profiles in ~/.aws/confi
      --version           version for aws-ssh

Use "aws-ssh [command] --help" for more information about a command.
```

### Environment variables

aws-ssh uses [viper](https://github.com/spf13/viper) under the hood, so it supports taking environment variables that correspond to the flags out of the box.

For example:

1. `AWS_SSH_DEBUG` is the `--debug` flag,
2. `AWS_SSH_NO_PROFILE_PREFIX` is `--no-profile-prefix`,
3. etc...

### Build

You'll need go>=1.16. Note that this project uses `go.mod`, so the project has to be cloned somewhere outside of the `GOPATH` directory.
Or just use provided `Dockerfile`.
