# Capsule

Capsule is a small, Linux-only container runtime experiment written in Go.

## Status

This project is in early design/prototyping.

## Goals

Capsule aims to:
- pull container images from a Docker registry
- run a single container from the CLI
- isolate workloads with Linux namespaces
- apply basic CPU and memory limits via cgroups

## Planned CLI

- `capsule pull <image>`
- `capsule run <image> [command...]`
- `capsule run --hostname <name> <image> [command...]`
- `capsule run --memory <bytes> --cpus <count> <image> [command...]`
- forwards stdin/stdout/stderr to the container process
- returns container exit code

## Roadmap

1. run a command in a child process
2. isolate hostname with UTS namespace
3. isolate root filesystem with `chroot`
4. isolate process tree and mounts with PID + mount namespaces
5. run rootless with user namespaces and UID/GID maps
6. apply CPU and memory limits with cgroups
7. pull image layers and config from registry
8. unpack layers and run pulled image

## Scope

- single-container runtime
- Linux only
- command-line only
- no daemon

## Non-goals

- orchestration or scheduling
- advanced networking
- production-hard security hardening
