<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://github.com/daytonaio/daytona/raw/main/assets/images/Daytona-logotype-white.png">
    <img alt="Daytona logo" src="https://github.com/daytonaio/daytona/raw/main/assets/images/Daytona-logotype-black.png" width="40%">
  </picture>
</div>

<br/>

<div align="center">

[![License](https://img.shields.io/badge/License-MIT-blue)](#license)
[![Go Report Card](https://goreportcard.com/badge/github.com/daytonaio/daytona)](https://goreportcard.com/report/github.com/daytonaio/daytona)
[![Issues - daytona](https://img.shields.io/github/issues/daytonaio/daytona)](https://github.com/daytonaio/daytona/issues)
![GitHub Release](https://img.shields.io/github/v/release/daytonaio/daytona)

</div>


<h1 align="center">Daytona Docker Provider</h1>
<div align="center">
This repository is the home of the <a href="https://github.com/daytonaio/daytona">Daytona</a> Docker Provider.
</div>
</br>

<p align="center">
  <a href="https://github.com/daytonaio/daytona-docker-provider/issues/new?assignees=&labels=bug&projects=&template=bug_report.md&title=%F0%9F%90%9B+Bug+Report%3A+">Report Bug</a>
    ·
  <a href="https://github.com/daytonaio/daytona-docker-provider/issues/new?assignees=&labels=enhancement&projects=&template=feature_request.md&title=%F0%9F%9A%80+Feature%3A+">Request Feature</a>
    ·
  <a href="https://join.slack.com/t/daytonacommunity/shared_invite/zt-273yohksh-Q5YSB5V7tnQzX2RoTARr7Q">Join Our Slack</a>
    ·
  <a href="https://twitter.com/Daytonaio">Twitter</a>
</p>

The Docker Provider allows Daytona to create workspace projects as Docker containers on your local or remote machine. It is a default provider in Daytona, which means it is installed with every server by default.

## Target Options

| Property                	| Type     	| Optional 	| DefaultValue                	| InputMasked 	| DisabledPredicate 	|
|-------------------------	|----------	|----------	|-----------------------------	|-------------	|-------------------	|
| Sock Path               	| String   	| true     	| /var/run/docker.sock        	| false       	|                   	|
| Remote Hostname         	| String   	| true     	|                             	| false       	| ^local$           	|
| Remote Port             	| Int      	| true     	| 22                          	| false       	| ^local$           	|
| Remote User             	| String   	| true     	|                             	| false       	| ^local$           	|
| Remote Password         	| String   	| true     	|                             	| true        	| ^local$           	|
| Remote Private Key Path 	| FilePath 	| true     	|                             	| false       	| ^local$           	|

### Default Targets

#### Local
| Property        	| Value                       	|
|-----------------	|-----------------------------	|
| Sock Path       	| /var/run/docker.sock        	|


## Code of Conduct

This project has adapted the Code of Conduct from the [Contributor Covenant](https://www.contributor-covenant.org/). For more information see the [Code of Conduct](CODE_OF_CONDUCT.md) or contact [codeofconduct@daytona.io.](mailto:codeofconduct@daytona.io) with any additional questions or comments.

## Contributing

The Daytona Docker Provider is Open Source under the [MIT License](LICENSE). If you would like to contribute to the software, you must:

1. Read the Developer Certificate of Origin Version 1.1 (https://developercertificate.org/)
2. Sign all commits to the Daytona Docker Provider project.

This ensures that users, distributors, and other contributors can rely on all the software related to Daytona being contributed under the terms of the [License](LICENSE). No contributions will be accepted without following this process.

Afterwards, navigate to the [contributing guide](CONTRIBUTING.md) to get started.

## Questions

For more information on how to use and develop Daytona, talk to us on
[Slack](https://join.slack.com/t/daytonacommunity/shared_invite/zt-273yohksh-Q5YSB5V7tnQzX2RoTARr7Q).