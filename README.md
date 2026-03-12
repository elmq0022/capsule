# Capsule

Capsule is a small, Linux-only container runtime experiment written in Go.

## Status

This project is in early design/prototyping.

## Goals

Capsule aims to:
- pull container images from a Docker registry
- run a single container from the CLI
- isolate workloads with Linux namespaces
- apply basic pid and memory limits via cgroups

## Planned CLI

- `capsule pull <image>`
- `capsule run <image> [command...]`
- `capsule run --hostname <name> <image> [command...]`
- `capsule run --memory <bytes> --cpus <count> <image> [command...]`
- rootfs data lives under `~/.local/share/capsule/rootfs/`
- forwards stdin/stdout/stderr to the container process
- returns container exit code

## Roadmap

- [x] run a command in a child process
- [x] run rootless with user namespaces and UID/GID maps
- [x] isolate hostname with UTS namespace
- [x] isolate root filesystem with `chroot`
- [x] isolate process tree and mounts with PID + mount namespaces
- [ ] isolate networking with a network namespace
- [x] apply pids and memory limits with cgroups
- [ ] pull image layers and config from registry
- [ ] unpack layers and run pulled image

## Scope

- single-container runtime
- Linux only
- command-line only
- no daemon
- no init process for PID 1

## Non-goals

These non goals are called out specifically to reduce project scope
and ensure completion of the stated goals.

- orchestration or scheduling
- advanced networking
- production-hard security hardening
- no mini init system for PID 1
