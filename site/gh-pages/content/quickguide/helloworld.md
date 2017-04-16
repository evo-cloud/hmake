---
title: Hello World
weight: 2
---
## A Simple Project

Let's make a simple _Hello World_ C++ project,
and see how HyperMake can help.

Here's the project directory layout:

```text
ProjectRoot
  |
  +--inc/
  +--src/
      +--hello.cpp
```

In _hello.cpp_:

```
#include <iostream>

int main(int argc, char *argv[]) {
    std::cout << "Hello World!" << std::endl;
    return 0;
}
```

To build it, use `g++ -o hello src/hello.cpp`.
Probably you will need to run `sudo apt install g++` first on Debian/Ubuntu if
you've never installed the toolchain.
Fortunately, using `HyperMake`, you don't need to worry about that on your host.

## The _HyperMake_ file

`HyperMake` is a YAML file which sits in the root directory of source tree.

```
---
format: hypermake.v0
name: hello
description: The Hello World Project
```

{{% notice tip %}}
The first line `---` indicates the start of a new document in YAML
{{% /notice %}}

- `format` is required and must be `hypermake.v0`.
  _hmake_ only accepts YAML files with `format: hypermake.v0`.
- `name` specifies the project name, which is required.
- `description` gives more information about the project. It's optional.

## Targets

Like _make_, the minimum execution unit is target:

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
```

The target _build_ is defined. It has three properties:

- `description`: a brief intro about what the target does;
- `image`: the docker image to create the container and run commands;
- `cmds`: a list of commands to execute inside the container.

Now, let's do

```
hmake build
./hello
```

It shows `Hello World`.

It always builds even if you don't have _g++_ installed on the host.

## Default

If typing `hmake build` is boring, defining default targets can simplify the
command as a single `hmake`.

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

settings:
  default-targets:
    - build
```

Now, typing `hmake` is equivalent to `hmake build`.

More settings can be added.
For example all the targets use the same image, it can be moved to `settings`.

```
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

All targets will have `image: 'gcc:4.9'` by default, unless explicitly specified
in target.

## Watches and Artifacts

Let's do some tricks here: `touch src/hello.cpp` or modify the file, and then
type `hmake`. You will see the `build` target is skipped:

```text
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

This is definitely not what we want.
However _hmake_ doesn't have the knowledge about source files and output files.
It must be explicitly specified using `watches` and/or `artifacts`:

```
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
matching a list of files/directories.
If the item is a directory, all sub-directories and files are watched recursively.
`artifacts` lists the output files.
Wildcard is not allowed here, and directory is not matched recursively.
_hmake_ rebuilds the target if any of the watched file is changed or
any artifact is missing.

## Dependencies

As the project is so simple that we can use an existing docker image `gcc:4.9` which
contains toolchain we need.
However in most cases, the public docker images are not always good enough, and
we want to install extra bits to build the project.
Then we need to build our own toolchain image.

Let's use `cmake` to build our project,
by adding `CMakeList.txt` under project root:

```
cmake_minimum_required(VERSION 2.8.0)
project(hello CXX)
include_directories("inc")
add_executable(hello src/hello.cpp)
```

Then we will need `cmake` in toolchain image, let's build one based on `gcc:4.9`.
Create a folder `toolchain` under project root and put a `Dockerfile` inside it:

```text
ProjectRoot
  |
  +--inc/
  +--src/
  | +--hello.cpp
  +--toolchain/
  | +--Dockerfile
  +--HyperMake
  +--CMakeList.txt
```

In `Dockerfile`:

```
FROM gcc:4.9
RUN apt-get update && apt-get install -y cmake && apt-get clean
```

And update `HyperMake`:

```
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

Target `toolchain` is added. Property `build` tells _hmake_ to build a
docker image from `toolchain/Dockerfile`.
And as `image: cmake-gcc:4.9` is specified in `settings`,
the built image will be `cmake-gcc:4.9`.

In target `build`, property `after` specifies `toolchain` must succeed
before `build` is able to run.

Now, type `hmake` and it will first run `toolchain` to build `cmake-gcc:4.9` and
the run `build` to call `cmake` to build the binary.
