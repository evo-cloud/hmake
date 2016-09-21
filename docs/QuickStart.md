# Quick Start Guide

This is a guide to write a HyperMake file for your project for the first time.

## Before Start

You need a project, of course. Let's make a simple _Hello World_ C++ project,
and see how HyperMake can help.

Here's the project directory layout:

```
ProjectRoot
  |
  +--inc/
  +--src/
      +--hello.cpp
```

In `hello.cpp`:

```cpp
#include <iostream>

int main(int argc, char *argv[]) {
    std::cout << "Hello World!" << std::endl;
    return 0;
}
```

To build it, use `g++ -o hello src/hello.cpp`. You will need toolchain installed.
Now, let's create a `HyperMake` to simplify the build.

## Create `HyperMake`

`HyperMake` can be composed in two ways:

- Wrapper mode: which wraps existing build tools, like GNU make
- Full mode: the native HyperMake format with all features.

For _Wrapper mode_, read details in [Wrapper Mode](WrapperMode.md).
In this guide, we will use _Full mode_.

Create a file called `HyperMake` under `ProjectRoot`.
It's a _YAML_ file, so let's start with:

```yaml
---
format: hypermake.v0
name: hello
description: The Hello World Project
```

The first line `---` is optional but recommended, as YAML parser will treat it
as the beginning of a new document.

`format` is required and must be assigned with `hypermake.v0`.
_hmake_ only parses YAML files with `format: hypermake.v0`.
`name` specifies the project name, which is required.
`description` gives more information about the project. It's optional.

## Adding targets

The most important part is the section defining targets:

```yaml
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
```

We defined one target above: `build`. It has three properties:

- `description`: a brief intro about what the target does;
- `image`: the docker image used to create the container and run commands;
- `cmds`: a list of commands to execute inside the container.

Now, we can use `hmake build` to build the project, and type

```
./hello
```

to show `Hello World`.

Under the hood, `hmake` creates a container temporarily, and maps current project
root to `/src` inside container and run the commands inside the container.

Are you feeling boring type `hmake build` every time? Why not just `hmake`?
Let's move on with default targets.

## Settings

The default targets can be specified inside `settings` section:

```yaml
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

settings:
  default-targets:
    - build
```

With `default-targets` specified in `settings`, we can type `hmake` without
arguments and it will run targets defined in `default-targets`.

The `settings` section defines properties which are common to all targets.
For example, we can define common properties for `docker`. Let's move `image`
property to settings:

```yaml
---
format: hypermake.v0
name: hello
description: The Hello World Project

targets:
  build:
    description: build hello binary
    cmds:
      - g++ -o hello src/hello.cpp

settings:
  default-targets:
    - build
  docker:
    image: 'gcc:4.9'
```

As we moved `image` to `settings/docker`, we can remove `images` from target
`build`. And all targets will have `image: 'gcc:4.9'` by default.

## Watches and Artifacts

Let's do some tricks here: `touch src/hello.cpp` or modify the file, and then
type `hmake`. You will see the `build` target is skipped:

```
HyperMake v1.1.0 https://github.com/evo-cloud/hmake

=> build 21:56:42.277
:] build
╔══════╤═══════╤════════╤════════════╤════════════╤═════╗
║Target│Result │Duration│Start       │Finish      │Error║
╠══════╪═══════╪════════╪════════════╪════════════╪═════╣
║build │Skipped│        │21:56:42.277│21:56:42.277│     ║
╚══════╧═══════╧════════╧════════════╧════════════╧═════╝
OK
```

This is definitely not what we want. The problem is _hmake_ doesn't know which
files are input and which are output. Let's tell _hmake_ by adding `watches` and
`artifacts` to target `build`:

```yaml
targets:
  build:
    description: build hello binary
    watches:
      - inc
      - src
    cmds:
      - g++ -o hello src/hello.cpp
    artifacts:
      - hello
```

The items in `watches` can be a path to a directory or a file, or with wildcards
matching a list of files/directories. If the item is a directory, all sub-directories
and files are watched recursively.
`artifacts` lists the output files. _hmake_ rebuilds the target if any of the
artifacts is missing. Wildcard is not allowed here, and directory is not matched
recursively.

Some targets doesn't require input files or generate output files. In this case
command line options can be used to explicitly rebuild the target: `-R`, `-r`, or
`-b`. See [Command Line](CommandLine.md) for details.

## Dependencies

As the project is so simple that we can use an existing docker image `gcc:4.9` which
contains toolchain we need.
However in most cases, the existing docker images are not always good enough, and
we want to install extra bits to build the project.
Then we need to build our own toolchain image.

Let's use `cmake` to build our project, by adding `CMakeList.txt` under project root:

```
cmake_minimum_required(VERSION 2.8.0)
project(hello CXX)
include_directories("inc")
add_executable(hello src/hello.cpp)
```

Then we will need `cmake` in toolchain image, let's build one based on `gcc:4.9`.
Create a folder `toolchain` under project root and put a `Dockerfile` inside it:

```
FROM gcc:4.9
RUN apt-get update && apt-get install -y cmake && apt-get clean
```

And update `HyperMake`:

```yaml
---
format: hypermake.v0
name: hello
description: The Hello World Project

targets:
  toolchain:
    description: build our own toolchain image
    watches:
      - toolchain
    build: toolchain

  build:
    description: build hello binary
    after:
      - toolchain
    watches:
      - inc
      - src
    cmds:
      - rm -fr rel && mkdir -p rel
      - cd rel && cmake .. && make
    artifacts:
      - rel/hello

settings:
  default-targets:
    - build
  docker:
    image: 'cmake-gcc:4.9'
```

Target `toolchain` is added, with property `build`, _hmake_ knows to build a
docker image, using `toolchain/Dockerfile`. And as `image: cmake-gcc:4.9` is specified
in `settings`, the built image will be `cmake-gcc:4.9`.

In target `build`, `after` specifies `toolchain` must succeed before `build` is able
to run. Because `build` will use the image `cmake-gcc:4.9` generated by `toolchain`.

Now, type `hmake` and it will first run `toolchain` to build `cmake-gcc:4.9` and
the run `build` to call `cmake` to build the binary.

## More

The above covers the basic features of _HyperMake_.
There are a lot more useful features.
Please read documents listed in [README](../README.md) for more details, and take
a look at `examples` for real samples.
