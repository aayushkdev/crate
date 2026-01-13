# crate

Crate is a small container runtime written in Go, built to explore containers work internally.
It supports both rootless (without sudo) and rootful (with sudo) execution, with rootless mode being the main focus.

---

## Getting started

Install using (Go 1.20+ recommended):
```bash
go install github.com/aayushkdev/crate/cmd/crate@latest
```
(ensure `GOBIN` is in path)

Verify installation:
```bash
crate --help
```


## Usage

### Pull an image

Pulls an image from a registry and stores it in the local image store.

```bash
crate pull alpine
```
If the image already exists locally, the pull is skipped.

---

### Create a container

Creates a container from an image and prints the container ID.

```bash
crate create alpine
```

---

### Start a container

Starts an existing container by ID.

```bash
crate start <CONTAINER_ID> [COMMAND] [ARG...]
```

Examples:

```bash
crate start c144672a8e04
```

```bash
crate start c144672a8e04 ls -l /
```

If no command is provided, the image’s default `CMD` is used.

---

### Run (create + start)

`run` is a convenience command that creates a new container and immediately starts it.

```bash
crate run alpine
```

```bash
crate run alpine /bin/sh -c "echo hello world"  
```


## Implemented Concepts

### Isolation

* PID namespace
* UTS namespace (hostname)
* Mount namespace
* User namespace (rootless mode)

### Filesystem

* Root filesystem setup using `pivot_root` (or `chroot` in rootless mode)
* `/proc` mounted inside the container
* `/dev` mounted as `tmpfs` with minimal devices (`null`, `zero`, `random`, `urandom`, `full`, `shm`, `pts`, `ptmx`) 
* `/run` mounted as `tmpfs`
* `/sys` mounted read-only in rootful mode

### Image handling

* Image name parsing (`repo:tag`)
* Pulling images from registries (docker only for now)
* OCI/Docker manifest resolution
* Local blob store (layers and config)
* Local image metadata cache (Prevents unecessary pulls)

### Process execution

* PID 1 replaced with the container process using `execve`
* Proper PATH-based command resolution (no shell)
* CMD, Entrypoint and environment variables used from image config


## Far off goals (for now)

* Better Process management
* Networking
* Cgroups / resource limits
* Volume mounts
* More configuration options
* Security hardening
* Full OCI spec compliance