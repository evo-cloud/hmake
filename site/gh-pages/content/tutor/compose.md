---
title: Docker Compose
weight: 5
---

_hmake_ supports [docker-compose](https://docs.docker.com/compose/overview/)
out-of-box, using property `compose`.

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

Please refer to [Background Target]({{< relref "targetexec.md#background-targets" >}}), [Docker Driver]({{< relref "references/dockerdrv.md" >}}) for details.
