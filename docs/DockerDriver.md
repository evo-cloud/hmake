# The _docker_ Execution Driver

This execution driver interprets commands or scripts and run inside the specified
docker container.

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

- `env-files`: list of files providing environment variables, see `--env-files`
  of `docker run`;

- `console`: when `true`, the current stdin/stdout/stderr is directly passed to
  command which is able to fully control the current console, equivalent to
  `docker run -it`.
  Default is false, equivalent to `docker run -a STDOUT -a STDERR`.

  _NOTE: When enabled, no output is captured or logged._

- `build`: path to `Dockerfile`, when specified, this target builds a docker
   image first. `image` property specifies the image name and tag.
   The value can point to a `Dockerfile` (e.g. `build: build/Dockerfile.arm`)
   which indicates the folder containing the file is the context folder.
   And the value can also point to a folder which contains a `Dockerfile`
   (e.g. `build: build`) which uses the folder as context folder and looks for
   `Dockerfile` there.

   It's strongly recommended to put `Dockerfile` and any related files to
   `watches` list;

- `build-from`: the path of context folder for `docker build`.
  Without this property, the path is derived from path of `Dockerfile` specified
  in `build`. Please note, the path must be direct/indirect parent of the
  `Dockerfile` as required by `docker build`;

- `build-args`: list of args, corresponding to `docker build` option;

- `image`: with `build` it's the image name and tag to build,
  without `build`, it's the image used to create the container;

- `tags`: a list of tags in addition to `image` when do `docker build`;

- `commit`: commit running container into new image. Support multiple tags. 
  Image tag will be 'latest', if not self-defined in image name. E.g.

  ```yaml
  target:
      image: new-image-name:newtag
      cmds:
          <Some commands>
      commit: 
            - new-image-name:tag1
            - new-image-name:tag2
  ```

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
- `net`: when specified, only allowed value is `host`, when specified, run
  container with `--net=host --uts=host`;
- `user`: passed to `docker run --user...`, by default, current `uid:gid` are 
  passed (with _docker-machine_ the `uid:gid` is queried from the virtual machine
  running docker daemon).
  It must be explicitly specified `root` if the script is executed as root
  inside container.
  When a non-root user is explicitly specified, all group IDs are automatically
  passed using `--group-add`l;
- `groups`: explicitly specify group IDs to pass into container, instead of
  passing all of them;
- `volumes`: a list of volume mappings passed to `-v` option of `docker run`.

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

_NOTE_

On Mac OS, only paths under `/Users` can be mapped into the container.
All project trees must sit under `/Users`.

On Windows, only paths under `C:\Users` can be mapped into the container.
All project trees must sit under `C:\Users`.

## User

By default _hmake_ uses current user (NOT root) to run inside container,
which make sure any file change has the same ownership as the environment outside.
If root is required, it can be explicitly specified `user: root`,
however, all files created inside container will be owned by `root` outside,
and you may eventually see some error messages like `permission denied` when you
do something outside.

## Limits

On Linux, _docker-machine_ is not supported, docker daemon must run on the same
host running _hmake_.
