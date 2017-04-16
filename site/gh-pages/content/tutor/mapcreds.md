---
title: Map Credentials
weight: 8
---
## Problem

Many CLI tools like docker, aws-cli, gcloud etc stores credentials in user's home
directory.
When build directly from the host, there's no problem accessing these services.
However, using _hmake_, build runs inside container which doesn't have the right
credentials, and will fail.

## Solution

Here's a commonly used practice to map credentials from local home directory.

```yaml
---
format: hypermake.v0
name: map-credentials

targets:
  build:
    description: build source code
    cmds:
      - ./build.sh
    env:
      - HOME=/tmp
    volumes:
      - '~/.ssh:/tmp/.ssh:ro'
      - '~/.aws:/tmp/.aws:ro'
```
