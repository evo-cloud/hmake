---
title: Push Docker Image
weight: 4
---

_hmake_ provides built-in support to push docker image to a remote registry.

```
---
format: hypermake.v0
name: push-sample

targets:
  build-image:
    description: build docker image
    build: Dockerfile
    image: 'myimage:tag'
    tags:
      - 'registry:5000/namespace/myimage:tag'
      - 'registry1/namespace/myimage:tag'

  push-image:
    description: push docker images
    after:
      - build-image
    push:
      - 'registry:5000/namespace/myimage:tag'
      - 'registry1/namespace/myimage:tag'
```

The property `push` specifies which images to push.
_hmake_ calls `docker push` locally to push the images,
so make sure the credentials are stored using `docker login` if the registry
requires authentication.
