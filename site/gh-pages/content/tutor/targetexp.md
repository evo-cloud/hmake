---
title: Target Expansion
weight: 9
---
It's quite common targets are defined similarly with very small difference.
For example, a build target which builds for different CPU architectures,
the only difference can be the name passing to the build script.

```yaml
---
format: hypermake.v0
name: expand-sample

targets:
  hmake-linux-amd64:
    description: static linked hmake binary for linux-amd64
    after:
      - vendor
    watches:
      - '**/**/*.go'
      - build.sh
    cmds:
      - ./build.sh linux amd64
    artifacts:
      - bin/linux/amd64/hmake
      - bin/hmake-linux-amd64.tar.gz
      - bin/hmake-linux-amd64.tar.gz.sha256sum

  hmake-linux-arm:
    description: static linked hmake binary for linux-arm
    after:
      - vendor
    watches:
      - '**/**/*.go'
      - build.sh
    cmds:
      - ./build.sh linux arm
    artifacts:
      - bin/linux/arm/hmake
      - bin/hmake-linux-arm.tar.gz
      - bin/hmake-linux-arm.tar.gz.sha256sum

  hmake-linux-arm64:
    description: static linked hmake binary for linux-arm64
    after:
      - vendor
    watches:
      - '**/**/*.go'
      - build.sh
    cmds:
      - ./build.sh linux arm64
    artifacts:
      - bin/linux/arm64/hmake
      - bin/hmake-linux-arm64.tar.gz
      - bin/hmake-linux-arm64.tar.gz.sha256sum
```

Obviously, there are a lot of duplications in above sample.
With target expansion, the above targets can be rewritten as

```
---
format: hypermake.v0
name: expand-sample

targets:
  hmake-linux-[arch:amd64,arm,arm64]:
    description: static linked hmake binary for linux-$[arch]
    after:
      - vendor
    watches:
      - '**/**/*.go'
      - build.sh
    cmds:
      - ./build.sh linux $[arch]
    artifacts:
      - bin/linux/$[arch]/hmake
      - bin/hmake-linux-$[arch].tar.gz
      - bin/hmake-linux-$[arch].tar.gz.sha256sum
```

It's possible to use multiple expansion variables.
