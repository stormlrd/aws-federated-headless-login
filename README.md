# Copyright Credit
Having dived into trying to get federated access going I've used myziyabo.co's blog and code as a baseline and heavilly modified it.

https://mziyabo.co/2022/aws-sso-headless-login/

Full credit goes to mziyabo for helping as a result as without their work this project would never have existed.

This heavilly modified version of the headless-sso has been renamed as a result and its use case has changed as I needed it for a python script I was making.

I've removed the .netrc file for example as I consider that a security risk.

The purpose of this version if you will is to be used within python scripts wanting to use aws cli calls looping through all the aws cli config profiles. My intention is to have the first federated login have a visible browser window for manual entry of your username + password + mfa combo for security purposes. Once thats been accomplished the authentication flow via the browser is about approvals only, not logging in; in full.

This is to only be used in conjunction with : https://github.com/stormlrd/aws-federated-python-awscli-skeleton as a result.

Have a look at that script as it will show you why this version exists.

Thanks myziyabo! You rock.

# Usage:
clone the repository and execute:
```
go build
```
you should get an aws-federated-headless-login executable built for use with aws-federated-python-awscli-skeleton python script template.

enjoy.