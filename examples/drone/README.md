# Example - Cross build Drone

This example demonstrates how to use HyperMake to cross build
[drone](https://github.com/drone/drone) for different platforms.

## Get Started

```sh
hmake -sv
```

## How it works

Simply a few steps:

- Build a docker image containing all tools
- Install dependencies (Go get)
- Git clone latest source code
- Go generate
- Go build for (in parallel)
  - linux-amd64
  - linux-arm64
  - linux-arm
  - windows-amd64
  - darwin-amd64

The output is under `release/OS/ARCH/drone`.

## Notes

The `go generate` step requires `sassc` which is built from source in the first
step building docker image. See `builder.Dockerfile` for details.
