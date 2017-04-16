---
title: Docker Driver
weight: 3
---
This execution driver interprets commands or scripts and run inside docker container
using specified image.
The driver uses `docker run` to start the container, not using Docker REST API,
so docker CLI is required.

## Properties

- `script`: a multi-line string represents a full script to execute inside the
  container;
  E.g.

  ```yaml
  targets:
      sample:
          script: |
              #!/bin/bash
              echo 'This is a bash script'
      sample1:
          script: |
              #!/usr/bin/env perl
              print "a perl script"
  ```

- `cmds`: when `script` is not specified, this is a list of commands to execute
  for the target; E.g.

  ```yaml
  targets:
      sample:
          cmds:
              - mkdir -p bin
              - gcc -o bin/hello hello.c
  ```

  the list of commands is merged to generate a shell script:

  ```sh
  #!/bin/sh
  set -e
  mkdir -p bin
  gcc -o bin/hello hello.c
  ```

- `env`: a list of environment variables (the form `NAME=VALUE`) to be used for
  execution (the `-e` option of `docker run`); E.g.

  ```yaml
  targets:
      sample:
          env:
              - ARCH=x86_64
              - OS=linux
              - RELEASE        # without =VALUE, the value is populated from
                               # current environment of hmake
  ```

{{% notice tip %}}
It's very useful to specify environment variable without a value.
It's possible to specify the value at the time invoking hmake, instead of hard-code
in hmake files.
{{% /notice %}}

- `env-files`: list of files providing environment variables, see `--env-files`
  of `docker run`;

- `console`: when `true`, the current stdin/stdout/stderr is directly passed to
  command which is able to fully control the current console, equivalent to
  `docker run -it`.
  Default is false, equivalent to `docker run -a STDOUT -a STDERR`.

{{% notice info %}}
When `console` is enabled, no output is captured or logged.
{{% /notice %}}

- `build`: path to `Dockerfile`, when specified, this target builds a docker
   image first. `image` property specifies the image name and tag.
   The value can point to a `Dockerfile` (e.g. `build: build/Dockerfile.arm`)
   which indicates the folder containing the file is the context folder.
   And the value can also point to a folder which contains a `Dockerfile`
   (e.g. `build: build`) which uses the folder as context folder and looks for
   `Dockerfile` there.

{{% notice tip %}}
`Dockerfile` and any related files should be included in `watches` list
{{% /notice %}}

- `build-from`: the path of context folder for `docker build`.
  Without this property, the path is derived from path of `Dockerfile` specified
  in `build`. Please note, the path must be direct/indirect parent of the
  `Dockerfile` as required by `docker build`;

- `build-args`: list of args, corresponding to `docker build` option;

- `image`: with `build` it's the image name and tag to build,
  without `build`, it's the image used to create the container;

- `tags`: a list of tags in addition to `image` when do `docker build`;

- `commit`: commit running container into new image. Support multiple tags.
  Image tag will be `latest`, if not self-defined in image name. E.g.

  ```yaml
  target:
      image: image-name:tag
      cmds:
          - Some commands
          - ...
      commit:
            - new-image-name:tag1
            - new-image-name:tag2
  ```

  It will first run the commands in the container created from _image-name:tag_
  and then commit the container to image _new-image-name:tag1_, _new-image-name:tag2_.
  It's the alternative way to build an image, versus using property `build`.

- `cache`: only used to specify `false` which adds `--no-cache` to `docker build`;
- `content-trust`: only used to specify `false` which adds
  `--disable-content-trust` to `docker build/run`;
- `src-volume`: the full path inside container where project root is mapped to.
  Default is `/src`;
- `expose-docker`: when set `true`, expose the host docker server connectivity
  into container to allow docker client run from inside the container.
  This is very useful when docker is required for build and avoid problematic
  docker-in-docker;
- `privileged`: run container in privileged mode, default is `false`;
- `net`: when specified, add `--net` option to docker CLI.
  When set to `host`, also add `--uts=host`;
- `link`: a list of strings of `container:hostname` mapping to `--link` option.
  This is very useful when combined with `compose` (docker-compose);
- `user`: passed to `docker run --user...`, by default, current `uid:gid` are
  passed (with _docker-machine_ the `uid:gid` is queried from the virtual machine
  running docker daemon).
  It must be explicitly specified `root` if the script is executed as root
  inside container.
  When a non-root user is explicitly specified, all group IDs are automatically
  passed using `--group-add`l;
- `groups`: explicitly specify group IDs to pass into container, instead of
  passing all of them;
- `volumes`: a list of volume mappings passed to `-v` option of `docker run`;
- `compose`: run `docker-compose`, see below for details.

The following properties directly maps to `docker build/run` options:

- `cap-add`, `cap-drop`
- `devices`
- `hosts`: mapped to `--add-host`
- `dns`, `dns-opts`, `dns-search`
- `blkio-weight`, `blkio-weight-devices`
- `device-read-bps`, `device-write-bps`, `device-read-iops`, `device-write-iops`
- `cpu-shares`, `cpu-period`, `cpu-quota`, `cpuset-cpus`, `cpuset-mems`
- `kernel-memory`, `memory`, `memory-swap`, `memory-swappiness`, `shm-size`
- `ulimit`
- `labels`, `label-files`
- `pull`, `force-rm`

All above properties can also be specified in `settings`/`local` under
`docker` section:

```yaml
settings:
    docker:
        property: value
```

## Volume Mapping

By default the current project root is mapped into container at `src-volume`,
default value is `/src`.
As the script is a shell script, the executable `/bin/sh` must be present in
the container.

The host side path is translated with the following rule:

- It's relative path, it's relative to working directory (property `workdir`);
- It's absolute path (including starting with `~/`, `~` expands to home), it's absolute on the host;
- It starts with `-/`, it's relative to project root.

Example:

```yaml
targets:
  volumes:
    - 'abc:/var/lib/abc'  # host path is $HMAKE_PROJECT_DIR/$HMAKE_TARGET_DIR/abc
    - '~/.ssh:/root/.ssh' # host path is $HOME/.ssh
    - '/var/lib:/var/lib' # host path is /var/lib
    - '-/src:/src'        # host path is $HMAKE_PROJECT_DIR/src
```

{{% notice info %}}
On Mac OS, only paths under `/Users` can be mapped into the container.
All project trees must sit under `/Users`.
{{% /notice %}}

{{% notice info %}}
On Windows, only paths under `C:\Users` can be mapped into the container.
All project trees must sit under `C:\Users`.
{{% /notice %}}

## User

By default _hmake_ uses current user (NOT root) to run inside container,
which make sure any file change has the same ownership as the environment outside.
If root is required, it can be explicitly specified `user: root`,
however, all files created inside container will be owned by `root` outside,
and you may eventually see some error messages like `permission denied` when you
do something outside.

## Docker Compose

The property `compose` is used to run `docker-compose` as a background target.
The value can be a single string pointing to a directory containing
`docker-compose.yml` (`docker-compose.yaml`) or a full path to a file with
alternative name instead of `docker-compose.yml`,
or an object containing detailed properties:

- `file`: the path to a directory containing `docker-compose.yml`, or to a file
  with alternative name;
- `project-name`: override project name (`--project-name`);
- `deps`: when `false`, add `--no-deps`;
- `recreate`: when `false`, add `--no-recreate`, or `force`, add `--force-recreate`;
- `build`: when `true`, add `--build`, or `false`, add `--no-build`;
- `remove-orphans`: when `true`, add `--remove-orphans`;
- `services`: a list of strings as service names after `docker-compose up` command line.

When `compose` is present, the target is executed as a background target.
`docker-compose up -d` is used to launch containers in the background.
If `cmds` or `build` are also present in the same target, they are executed after
`docker-compose` launched the containers.

Other targets can take dependency on a background target (e.g. with `compose`), and
in this case, use `net` and `link` to connect target to containers launched by
`docker-compose`. This is very useful to launch a testing environment and run
test code from targets.

## Known Issues

- Docker machine backed by VirtualBox: Docker for Mac is recommended instead of VirtualBox
    - Unstable NAT service: the NAT service from VirtualBox will eventually disconnect;
    - Hard link not supported: hard links can't be created on mapped volumes;
- Time drifting inside the VM: with Docker for Mac, the time may drift because there's
  no time synchronization at current stage;
- Memory exhaust: observed with Docker for Mac, restart Docker service solve the problem;
- Docker `--label` bug: Docker _1.12_ has a bug parsing command line `--label` which works
  with Docker _1.11_, putting labels inside `Dockerfile` works fine.

## Limits

On Linux, _docker-machine_ is not supported, docker daemon must run on the same
host running _hmake_.
