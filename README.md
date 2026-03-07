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

- [x] run a command in a child process
- [ ] isolate hostname with UTS namespace
- [ ] isolate root filesystem with `chroot`
- [ ] isolate process tree and mounts with PID + mount namespaces
- [ ] run rootless with user namespaces and UID/GID maps
- [ ] apply CPU and memory limits with cgroups
- [ ] pull image layers and config from registry
- [ ] unpack layers and run pulled image

## Scope

- single-container runtime
- Linux only
- command-line only
- no daemon

## Non-goals

- orchestration or scheduling
- advanced networking
- production-hard security hardening
