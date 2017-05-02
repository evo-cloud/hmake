---
title: Installation
weight: 1
---
## Install Docker

Install [Docker](https://www.docker.com), version *1.12.3* or above is required.

Run

```sh
docker version
```

and make sure it shows both client and server versions correctly.

_NOTE:_ Docker machine is no longer supported.
For Mac/Windows, please use Docker for Mac or Docker for Windows.

## Docker Compose (Optional)

Install [docker-compose](https://docs.docker.com/compose/install/) if you want to
use `compose` property in targets.

## Install HyperMake

### For Mac

```sh
brew tap evo-cloud/toolkit  # only do this once
brew install hmake
```

Or download and extract the binary directly

```sh
curl -s https://github.com/evo-cloud/hmake/releases/download/v1.3.1/hmake-darwin-amd64.tar.gz | sudo tar -C /usr/local/bin -zx
```

### For Linux

```sh
curl -s https://github.com/evo-cloud/hmake/releases/download/v1.3.1/hmake-linux-amd64.tar.gz | sudo tar -C /usr/local/bin -zx
```

### For Windows

Download and extract the binary from

```text
https://github.com/evo-cloud/hmake/releases/download/v1.3.1/hmake-windows-amd64.zip
```

## Anything else?

No. That's all you need. Enjoy!
