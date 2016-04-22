# HyperMake

HyperMake helps you build projects without installing pre-requisites in your
local environment.
It uses containers to build projects, all pre-requisites can be pre-installed
inside the container.

A similar project is [drone](http://readme.drone.io) which is built to be a service.
While HyperMake is built as a handy tool with a few special features:

- Brings in the _target_ concept from traditional _GNU make_,
  targets can be defined in one HyperMake file or multiple,
  and targets has dependencies, can be built selectively.
- Concurrent builds, targets without explicit dependencies can be built concurrently.
- Synchronized execution, the execution of individual command can be synchronized
  across targets being built concurrently with help of _macros_

## Getting Started

Download the binary from github release and place it in a folder which can be
searched in `PATH`.

Or if you know Go well enough

```
go get github.com/evo-cloud/hmake
```

_hmake_ expects a `HyperMake` file in root of the project,
and you can run `hmake` from any sub-directory inside the project,
the command will figure out project root by locating `HyperMake` file.

In any sub-directory, a file called `HyperMake` or `*.hmake` will be collected
and aggregated into the _world view_ of _hmake_.

## File Format

In `HyperMake` or `*.hmake`, define any of the following things:

- Targets: the target to build, including dependencies and commands.
- Settings: the settings applies to _hmake_.

Here's the schema in example:

```yaml
---
format: hypermake.v0 # this indicates this is a HyperMake file

targets:
    all:
        after:
            - web
            - server
    web:
        image: evo-cloud/gobuilder:1.1
        envs:
            - ENV       # this requires a pre-defined environment
            - APP=web   # this defines an environment variable
        cmds:
            - make $TARGET  # TARGET is pre-defined by hmake
    server:
        # image not specified, using that in configurations
        cmds:
            - make $TARGET

settings:
    default-image: evo-cloud/gobuilder:1
    default-shell: ["/bin/bash", "-c"]
    hmake-dir: .hmake
    mapped-path: /root/src
    volumes:
        - local:inside-path
        - ...
    envs:
        - ...
    map-docker: true # or inside path of docker unix socket
    privileged: true
    caps-add:
        - ...
    caps-drop:
        - ...

```

#### Rules for settings

The settings in files in parent folder will be visible to targets in the
same file and files in sub-directories.
While settings doesn't populate back to targets defines in parent folders.

There's one special file called `hmakerc` which contains only settings and
applies to all _hmake_ targets defined in current folder and sub-directories.

The current schema of settings is a flat map, with values be of arbitrary types

#### Pre-defined Environment Variables
