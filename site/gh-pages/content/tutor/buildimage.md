---
title: Build Docker Image
weight: 3
---
It's simple and straight-forward to build Docker images using _hmake_.

## Use Docker build

```yaml
---
format: hypermake.v0
name: build-sample

targets:
  build-image:
    description: build docker image
    build: dir-contains-Dockerfile
    image: 'my-image:tag'
    tags:
      - 'my-image1:tag1'
      - 'my-image2:tag2'
```

Specify property `build` to run `docker build` for the target.
The value can be a path to a directory containing `Dockerfile`,
or path including filename.
If filename is included, it's not necessary to be `Dockerfile`,
_hmake_ will take care of the right command line arguments to `docker build`.

The `image` property specifies the final image name,
additional tags can be specified using `tags` property.

For more properties, please refer to
[Docker Driver]({{< relref "references/dockerdrv.md" >}})
reference for details.

## Use Docker Commit

This is the alternate approach of building a docker image without using
`docker build` or `Dockerfile`.
It's a normal target with `commit` property.

```yaml
---
format: hypermake.v0
name: build-sample1

targets:
  build-image:
    description: build docker image
    image: base-image
    cmds:
      - ./setup.sh
    commit: 'my-image:tag'  
```

First, as a normal target, it runs `./setup.sh` inside container.
The script may install software or modify the file system of the container
created from `base-image`.
When the script finishes, the container is committed to image `my-image:tag`
as specified by property `commit`.

The value of `commit` can be a single image name or a list of image names, like:

```yaml
---
format: hypermake.v0
name: build-sample1

targets:
  build-image:
    description: build docker image
    image: base-image
    cmds:
      - ./setup.sh
    commit:
      - 'my-image:tag'  
      - 'my-image1:tag1'
      - 'my-image2:tag2'
```

## As Base Image

The image built can be used as base image for other targets.
For most projects, a common practice is create a toolchain image and used by
all other targets.

```yaml
---
format: hypermake.v0
name: common-project

targets:
  toolchain:
    description: build toolchain image
    build: toolchain

  build:
    description: build from source
    after:
      - toolchain
    cmds:
      - ./build.sh

  test:
    description: test the build
    after:
      - build
    cmds:
      - ./test.sh

settings:
  docker:
    image: mytoolchain
```
