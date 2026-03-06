# Capsule Design Document

## Overview

## Goals

Write a program that:
- pulls a container image from a Docker registry
- runs the container
- limits available CPU and memory resources
- runs the container in its own isolated namespace

## Scope

- single-container runtime
- Linux only
- command-line only
- no daemon
- no orchestration

## Non-Goals

- multi-container scheduling
- overlay filesystem support
- production-hard security hardening
- advanced networking

## CLI Contract

- `capsule run <image> [command...]`
- `capsule pull <image>`
- `capsule run --hostname <name> <image> [command...]`
- `capsule run --memory <bytes> --cpus <count> <image> [command...]`
- forwards stdin/stdout/stderr to the container process
- returns container exit code

## Milestones

1. run a command in a child process
2. isolate hostname with UTS namespace
3. isolate root filesystem with `chroot`
4. isolate process tree and mounts with PID + mount namespaces
5. run rootless with user namespaces and UID/GID maps
6. apply CPU and memory limits with cgroups
7. pull image layers and config from registry
8. unpack layers and run pulled image

## Architecture

- CLI parser
- image reference parser
- registry client
- layer downloader
- layer unpacker
- rootfs builder
- namespace launcher
- cgroup manager
- process supervisor

## Run Flow

1. parse flags and image reference
2. ensure image exists locally or pull it
3. build container rootfs from layers
4. create namespaces and user mappings
5. configure cgroups
6. `chroot` into rootfs
7. exec target command
8. forward signals and wait for exit
9. clean up mounts and temp paths

## Filesystem Layout

- image metadata under local state directory
- compressed layers under local cache directory
- unpacked rootfs under per-image directory
- runtime bundle under per-container temp directory

## Security Assumptions

- local development only
- untrusted workloads are out of scope
- rootless mode preferred when supported
- no seccomp/apparmor policy in initial version

## Test Plan

- run a command and verify exit code passthrough
- set hostname and verify isolation from host
- run `ps` and verify isolated process list
- verify host user maps to root in container only
- apply memory limit and verify OOM behavior
- apply CPU limit and verify throttling
- pull `alpine` and run `/bin/sh -c "echo ok"`

## Error Handling

- concise user-facing errors
- debug logs for pull/unpack/namespace setup
- fail fast on setup errors
- best-effort cleanup on failure
