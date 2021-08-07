### aws-ssh is your swiss knife for SSH into AWS instances

This program goes through all available AWS accounts in parallel and determines

IP addresses of ec2 instances. It also detects so-called "bastion" instances.

### Installation

1. For MacOS use homebrew and install with `brew install springload/tools/aws-ssh`
2. For Arch Linux install from AUR: https://aur.archlinux.org/packages/aws-ssh/
3. For other Linux distributions grab either .deb or .rpm from GitHub releases https://github.com/springload/aws-ssh/releases
4. Otherwise, just get the binary from the releases as above (one of those .tar.gz files), unpack it and install somewhere in your PATH.

### Quickstart

After the installation, this tool requries AWS CLI access to be configured. To set it up, please refer to the official documentation from AWS.
After you have at least one AWS profile configured, run `aws-ssh test` to see that everything is working correctly.

### Use ec2 connect feature to SSH to instances without managing SSH keys.

If your AWS EC2 instances are set up for ec2 connect and your AWS user has appropriate IAM policies, aws-ssh can connect to the instance straight away.

There are certain prerequisites:

1. Check `--ssh-config-path` option of "aws-ssh connect". aws-ssh will generate an config for SSH under this path, which will have the instance IP address, user to log under and even config for the bastion hosts. This file will be rewritted on every run of aws-ssh
2. Include the above file into your ssh config using `Include ec2_connect_config` where `ec2_connect_config` is the filename (or path) as above.
3. You can specify AWS profile from your config using `-p` flag and the instance id using `-i` flag.
4. But it's boring to look up the instance id every time so you can run `aws-ssh update` to generate cache of all EC2 instances across all available AWS profiles
5. Then just run `aws-ssh connect` to search for the right instance and press "Enter"

### Use reconf feature

Instead of using EC2 connect, one can have their ssh keys directly on the instances, so for those cases there is `aws-ssh reconf` command which just generates ssh config to be included in the main one.

### EC2 instance configuration tags

There are the following EC2 instance tags that change behaviour:

1. (Deprecated) If a bastion instance has tag "Global" with value "yes", "true" or "1", then aws-ssh will use it for all VPCs. If there are multiple bastion instances, it chooses the instance that has the most common match in name.
2. "x-aws-ssh-global" - same as the above
3. "x-aws-ssh-user" - sets the ssh username in the config.
4. "x-aws-ssh-port" - sets the ssh port in the config.

#### Additional ~/.aws/config properties

You can add an additonal property to AWS profiles like

```ini

[profile your_profile]
...
aws-ssh-domain = domain.com
```

To have the domain appended to the instance name, so in the SSH config it becomes `{profile}.{instance_name}.{domain}`

### Environment variables

aws-ssh uses [viper](https://github.com/spf13/viper) under the hood, so it supports taking environment variables that correspond to the flags out of the box.

For example:

1. `AWS_SSH_DEBUG` is the `--debug` flag,
2. `AWS_SSH_NO_PROFILE_PREFIX` is `--no-profile-prefix`,
3. etc...

Basically, take any flag, add `AWS_SSH_` prefix, uppercase it and replace "-" with "\_".

### Build manually and contribute

You'll need go>=1.16. Note that this project uses `go.mod`, so the project has to be cloned somewhere outside of the `GOPATH` directory.
Or just use provided `Dockerfile`.
