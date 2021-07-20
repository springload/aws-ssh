### aws-ssh is your swiss knife for SSH into AWS instances

This program goes through all available AWS accounts in parallel and determines

IP addresses of ec2 instances. It also detects so-called "bastion" instances.

There are the following EC2 instance tags that change behaviour:

1. (Deprecated) If a bastion instance has tag "Global" with value "yes", "true" or "1", then aws-ssh will use it for all VPCs. If there are multiple bastion instances, it chooses the instance that has the most common match in name.
2. "x-aws-ssh-global" - same as the above
3. "x-aws-ssh-user" - sets the ssh username in the config.
4. "x-aws-ssh-port" - sets the ssh port in the config.

### Utilise ec2 connect feature

If your AWS EC2 instances are set up for ec2 connect and your AWS user has appropriate IAM policies, aws-ssh can connect to the instance straight away.

There are certain prerequisites:

1. Check `--ssh-config-path` option of "aws-ssh connect". aws-ssh will generate an config for SSH under this path, which will have the instance IP address, user to log under and even config for the bastion hosts. This file will be rewritted on every run of aws-ssh
2. Include the above file into your ssh config using `Include ec2_connect_config` where `ec2_connect_config` is the filename (or path) as above.
3. You can specify AWS profile from your config using `-p` flag and the instance id using `-i` flag.
4. But it's boring to look up the instance id every time so you can run `aws-ssh update` to generate cache of all EC2 instances across all available AWS profiles
5. Then just run `aws-ssh connect` to search for the right instance and press "Enter"

### Utilise reconf feature

Instead of using EC2 connect, one can have their ssh keys directly on the instances, so for those cases there is `aws-ssh reconf` command which just generates ssh config to be included in the main one.

### Basic usage

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

Basically, take any flag, add `AWS_SSH_` prefix, uppercase it and replace "-" with "\_".

### Build

You'll need go>=1.16. Note that this project uses `go.mod`, so the project has to be cloned somewhere outside of the `GOPATH` directory.
Or just use provided `Dockerfile`.
