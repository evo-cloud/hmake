---
title: Target Execution
weight: 2
---
For full specification of all properties, please refer to

- [File Format]({{< relref "references/fileformat.md" >}})
- [Docker Driver]({{< relref "references/dockerdrv.md" >}})

## How Target Executes

A target always executes inside Docker container.

```
---
format: hypermake.v0
name: hello
description: The Hello World Project

targets:
  build:
    description: build hello binary
    image: 'gcc:4.9'
    cmds:
      - g++ -o hello src/hello.cpp
      - strip hello
```

In the above example, a container is created using image `gcc:4.9` as specified
by property `image`.
And lines in `cmds` are merged into a shell script `.hmake/build.sh`:

```sh
#!/bin/sh
set -e
g++ -o hello src/hello.cpp
strip hello
```

Finally, the execution is equivalent to invoke

```sh
docker run -a STDOUT -a STDERR -v `pwd`:/src -w /src gcc:4.9 /bin/sh .hmake/build.sh
```

## Source Tree Mapping

As targets are executed inside containers, the complete source tree of the project
is mapped into the container read-write
(details [here]({{< relref "references/dockerdrv.md#volume-mapping" >}})).
So all source files can be accessed inside container, and modifications in the
source tree is actually on the host.

The default path inside the container is `/src` and can be overridden using
property `src-volume` in target.

All file/path references are restricted inside project source tree
(except docker volume mapping `volumes` property in targets).

## Execution Order

Targets are executed according to the dependencies defined using `before` and
`after`.
When possible, _hmake_ will execute targets in parallel if dependencies are
satisfied.

If any target fails, _hmake_ will wait until all running targets finishes and exit.
It fails fast, and won't continue other ready-to-run targets.

## Watches and Artifacts

_hmake_ has no knowledge about input/output of a target.
Though in certain situation like build docker image, the execution driver
is able to figure out the output, for most targets, it's impossible for _hmake_
to find out automatically.

To rebuild the target only on relevant changes,
properties `watches` and `artifacts` are introduced to explicitly specify what
are the inputs and what are the outputs.

```
---
format: hypermake.v0
name: hello
description: The Hello World Project

targets:
  build:
    description: build hello binary
    image: 'gcc:4.9'
    watches:
      - inc
      - src/**/**/*.cpp
      - !inc/**/**/*.hpp
    cmds:
      - g++ -o hello src/hello.cpp
    artifacts:
      - hello
```

Property `watches` specifies a list of source files and accepts
[wildcard]({{< relref "references/fileformat.md#path-wildcard" >}})
When the item is a directory, all files and sub-directories are watched
recursively.
With `!` prefixed, the item specifies files/directories to be excluded.

Property `artifacts` specifies the expected output files/directories of the target.
Unlike `watches`, wildcard is not allowed, and if the item is a directory,
it's not scanned for files and sub-directories underneath.

With these information, _hmake_ will track the mtime of watched files,
and determine whether the target can be simply skipped if no change was made
and artifacts are all available.

Without these information, or the information is not properly specified,
_hmake_ may incorrectly skip the target even some change was made.

Sometimes, certain targets must always be built regardless of changes.
In this case, specifying property `always` to `true` forces _hmake_ build
the target every time.
This is especially useful for targets running tests, lint, etc.

```
---
format: hypermake.v0
name: hello
description: The Hello World Project

targets:
  test:
    description: run test
    always: true
    cmds:
      - ./test.sh

  lint:
    description: run lint
    always: true
    cmds:
      - ./run_lint.sh
```


## Background Targets

Background targets are those targets which spawn processes and keep them running
in the background.
It's very useful to spin up a testing environment with a few background targets
and one of the test target runs testing code against these background targets.

The background targets are implemented using
[docker-compose](https://docs.docker.com/compose/overview/).

E.g.

```yaml
---
format: hypermake.v0
name: compose-sample

targets:
  build:
    description: build from source code
    cmds:
      - ./build.sh
    artifacts:
      - out/service/Dockerfile
      - out/service/service.bin
      - out/service-compose/docker-compose.yml

  pack:
    description: pack as docker image
    after:
      - build
    build: out/service
    image: 'myservice:latest'

  start:
    description: start built service in background
    after:
      - pack
    compose: out/service-compose

  test:
    description: test against service
    after:
      - start
    link:
      - 'service:service'
    cmds:
      - ./test.sh
```

In the above example, target `build` builds from source code and generates
_Dockerfile_ for pack, and _docker-compose.yml_ for run.
Target `pack` creates a docker image using output from `build`.
Target `start` spawns the built service in the background using `compose`.
And `test` runs tests against the started service.
