[![Build Status](https://travis-ci.org/evo-cloud/hmake.svg?branch=master)](https://travis-ci.org/evo-cloud/hmake)

# HyperMake

It's a build tool which builds projects **without pre-requisites**.

![](https://raw.githubusercontent.com/evo-cloud/hmake/master/site/gh-pages/static/images/arm-hello.gif)

_Pains_

- Preparing building environment may take days
- Solving conflicts and incorrect versions of dependencies is painful
- Writing long and complicated building instructions when shipping the work

_Heals_

Forget about environment setup, what needed are only

- A running [docker](https://www.docker.com)
- `hmake` executable in `PATH`

HyperMake helps you build projects instantly and consistently without installing
pre-requisites in your local environment.
It uses containers to build projects, all pre-requisites are installed cleanly
and consistently inside the container.

Please read [Use Cases](docs/UseCases.md) to find out how HyperMake helps.

_Features_

- Brings back the experience of _make_
- Selectively build targets on demand
- Build in parallel

_HyperMake Server_

[HyperMake Server](https://github.com/evo-cloud/hmaked) is a CI/CD solution for
projects built with HyperMake.
It provides consistent environments for development, testing, integration and
deployment.

## Getting Started

Knowledge required as a user:

- Docker: [http://www.docker.com](http://www.docker.com)
- Very basic Unix shell and command line tools

As an author of HyperMake files:

- YAML: [http://yaml.org](http://yaml.org)

### Installation

Assume [Docker](http://www.docker.com) is already installed, and make sure it's
running properly.

_TIPS_

> When using `docker-machine`, many people encountered the issue docker complains
> unable to connect to docker daemon. The cause is the environment variables are
> not populated properly in current shell. Type the following commands:
>
```sh
# if you are using bash
eval $(docker-machine env MACHINE-NAME)
docker version
```
>
>Make sure `docker version` is able to show both versions of client and server,
>otherwise, docker may not work properly.

Now we can move on install `hmake`:

On Mac OS X, using Homebrew is the simplest way

```sh
brew tap evo-cloud/toolkit  # only do this once
brew install hmake
```

Alternatively, download from Github [releases](https://github.com/evo-cloud/hmake/releases)

```
curl -s https://github.com/evo-cloud/hmake/releases/download/v1.3.0/hmake-linux-amd64.tar.gz | sudo tar -C /usr/local/bin -zx
```

If you are on Mac OS, change `linux` above to `darwin`.
For Windows, change `linux` to `windows` and unpack the binary to some folder in
`%PATH%`.

Now do `hmake --version` to verify if it's properly installed.

### Do Something Funny

For the first time using hmake, let's do something funny - cross compile Linux
kernel without installing anything, even on Mac OS/Windows!

Checkout the examples in [hmake](https://github.com/evo-cloud/hmake) repository

```sh
git clone https://github.com/evo-cloud/hmake
cd hmake/examples/linux
hmake -sv
```

That's it! You get Linux kernel for both x86_64 and ARMv7 (vexpress board) in
a while.

See [README](examples/linux/README.md) for details.

## How It Works

_hmake_ works in a very simple way by running the commands of each target inside
the specified Docker container which already has pre-requisites installed.
The root directory of project tree is mapped into the container under a certain
path which can be customized, and the commands is able to access files inside
the project and can also produce output files into the project tree.

### State Directory

_hmake_ creates a state directory `$HMAKE_PROJECT_DIR/.hmake`
(see [File Format](docs/FileFormat.md) for the details of environment variables)
to store logs and state files.
The output (stdout and stderr combined) of each target is stored in files `TARGET.log`.
Debug log is stored as `hmake.debug.log`.
Summary file is stored as `hmake.summary.json`.

## Documents

Please read the following documents if more detailed information is needed

- [Quick Start](docs/QuickStart.md) is a step-by-step guide to write your first
  HyperMake file for your project;
- References are list of specifications including
  - [File Format](docs/FileFormat.md) defines the format of _hmake_ files;
  - [Command line](docs/CommandLine.md) specification;
- [Contributing](docs/Contribute.md) is a guideline for people who want to
  contribute to this project.
- Examples are always helpful
  - [Wrap Hello World for ARM](examples/arm-hello/README.md)
  - [Cross Compile Linux kernel](examples/linux/README.md)
- [FAQ and Best Practices](docs/FAQ.md)

## Supported Platform and Software

- docker 1.9 and above (1.9 - 1.11 tested)
- Linux (Ubuntu 14.04 tested)
- Mac OS X 10.9 and above (10.9, 10.11 tested)
- Windows 7 SP1

#### Limits

- On Mac OS X, the project tree must be under `/Users`;
- On Windows, the project tree must be under `C:\Users`;
- `docker-machine` is required on Mac OS X and Windows;
- `docker-machine` is not supported on Linux;

See [Docker Driver](docs/DockerDriver.md) for details.

#### Issues

If you meet any issues or have specific problems, please check
[FAQ and Best Practices](docs/FAQ.md) if there's already a solution.
Feel free to email the MAINTAINERS for any questions.

## License

MIT
