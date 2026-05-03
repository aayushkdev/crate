# crate

Crate is a small daemonless container runtime written in Go for Linux.
It launches containers directly, persists runtime state on disk instead of relying on a long-lived background daemon, and supports both rootless and rootful execution.

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
Crate resolves the tag on each pull and skips blob work when the resolved manifest digest is already present locally.

---

### List images

Lists local images from the manifest-backed metadata store.

```bash
crate images
```

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

Add `-d` / `--detach` to start it in the background.

Examples:

```bash
crate start c144672a8e04
```

```bash
crate start c144672a8e04 ls -l /
```

```bash
crate start -d c144672a8e04
```

If no command is provided, the image’s default `CMD` is used.

In attached mode, Crate allocates a real PTY so interactive shells and terminal programs behave normally.

---

### Run (create + start)

`run` is a convenience command that creates a new container and immediately starts it.

```bash
crate run alpine
```

```bash
crate run alpine /bin/sh -c "echo hello world"  
```

```bash
crate run -d alpine
```

---

### Stop a container

Stops one or more running containers by ID.

```bash
crate stop <CONTAINER_ID>
```

---

### Remove containers

Removes one or more stopped containers.

```bash
crate rm <CONTAINER_ID>
```

Running containers must be stopped first.

---

### List containers

Lists running containers by default.

```bash
crate ps
```

Show all containers:

```bash
crate ps -a
```

---

### View logs

Prints a container’s captured stdout/stderr.

```bash
crate logs <CONTAINER_ID>
```

Follow output:

```bash
crate logs -f <CONTAINER_ID>
```

---

### Remove images

Removes one or more local image tags.

```bash
crate rmi alpine:latest
```

If removing a tag leaves a manifest untagged, Crate deletes that manifest metadata and prunes any config or layer blobs that are no longer referenced by another local image.


## Implemented Concepts

### Isolation

* PID namespace
* UTS namespace (hostname)
* Mount namespace
* User namespace (rootless mode)
* Network namespace

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
* Manifest-based local image metadata with mutable local tags
* Local blob store (layers and config)
* Local image listing and removal
* Blob pruning when untagged manifests become unreferenced

### Process execution

* PID 1 replaced with the container process using `execve`
* Proper PATH-based command resolution (no shell)
* CMD, Entrypoint and environment variables used from image config
* PTY-backed attached mode for interactive shells and terminal programs
* Container lifecycle commands: `start`, `stop`, `ps`, `logs`, `rm`, and detached mode

### Networking

* Host, private, and disabled networking modes
* Private network namespaces created with `CLONE_NEWNET`
* Rootless private networking via `pasta`
* Loopback brought up inside isolated network namespaces
* `/etc/hosts` and `/etc/resolv.conf` copied into the container rootfs
* Parent/child synchronization so workloads start after network setup
* Automatic fallback from private networking to disabled networking when `pasta` is unavailable
* Network helper lifecycle tracking and teardown


## Far off goals (for now)

* Cgroups / resource limits
* Volume mounts
* More configuration options
* Security hardening
* Full OCI spec compliance

## Notes

In rootless mode, privilege-drop flows inside the container do not work. Switching from container root to another user/group after startup with tools like `setpriv`, `su`, or similar mechanisms fails because unprivileged GID mapping requires disabling `setgroups(2)`.
