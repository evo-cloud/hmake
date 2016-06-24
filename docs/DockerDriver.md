# The _docker_ Execution Driver

This driver generates the same script as [shell driver](ShellDriver.md) but run
it inside a docker container. The following properties are supported in
additional to properties supported by _shell_ driver:

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
- `cache`: only used to specify `false` which adds `--no-cache` to `docker build`;
- `content-trust`: only used to specify `false` which adds
  `--disable-content-trust` to `docker build/run`;
- `src-volume`: the full path inside container where project root is mapped to.
  Default is `/root/src`;
- `expose-docker`: when set `true`, expose the host docker server connectivity
  into container to allow docker client run from inside the container.
  This is very useful when docker is required for build and avoid problematic
  docker-in-docker;
- `env`: list environment variables passed to container, can be `NAME=VALUE` or
  `NAME` and maps to `-e` option of `docker run`;
- `env-files`: list of files providing environment variables, see `--env-files`
   of `docker run`;
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

The `console` property is also supported by `docker` driver which emits `-it` option
to docker client instead of `-a STDOUT -a STDERR`. And similarly to _shell_ driver,
no output is captured or logged.

## Volume Mapping

By default the current project root is mapped into container at `src-volume`,
default value is `/root/src`.
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
