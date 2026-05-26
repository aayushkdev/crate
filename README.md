# crate

Crate is a small daemonless container runtime written in Go for Linux.
It launches containers directly, persists runtime state on disk instead of relying on a long-lived background daemon, and supports both rootless and rootful execution.

---

## Getting started

Requires Go 1.20+.

```bash
git clone https://github.com/aayushkdev/crate.git
cd crate
go build -o crate ./cmd/crate/
mv crate ~/.local/share/bin/   # or any directory in your PATH
```

Or directly:

```bash
go install github.com/aayushkdev/crate/cmd/crate@latest
```

(ensure `GOBIN` is in your `PATH`)

## Usage

| Command | Description |
|---------|-------------|
| `crate pull <image>` | Pull an image from a registry |
| `crate images` | List local images |
| `crate run [--name] [--user] [-n] [-p] [-d] <image> [cmd]` | Create and start a container |
| `crate create [--name] [--user] [-n] [-p] <image>` | Create a container (without starting) |
| `crate start <container> [cmd]` | Start an existing container |
| `crate ps [-a]` | List running containers (use `-a` for all) |
| `crate stop <container>...` | Stop running containers |
| `crate rm <container>...` | Remove stopped containers |
| `crate container prune` | Remove all stopped containers |
| `crate logs [-f] <container>` | View container logs |
| `crate rmi <image>...` | Remove local image tags |
| `crate image prune` | Remove unreferenced image data |



Containers can be referenced by name or ID. Crate resolves names by scanning container configs for a matching `Name` before falling back to hex-ID matching.

Typical workflow:

```bash
crate pull nginx
crate run --name my-nginx -d -n private -p 8080:80 nginx
crate ps
crate logs my-nginx
crate stop my-nginx
crate rm my-nginx
```

Flags:

| Command | Flag | Description |
|---------|------|-------------|
| `run` | `-d, --detach` | Run the container in the background |
| `run`, `create` | `--name <name>` | Assign a container name |
| `run`, `create` | `--user <user>` | Run as `USER`, `UID`, `USER:GROUP`, or `UID:GID` |
| `run`, `start` | `-w, --workdir <path>` | Override the working directory for that launch |
| `run`, `create` | `-n, --network <mode>` | Select `host`, `none`, or `private` networking |
| `run`, `create` | `-p, --publish <spec>` | Publish a port with `HOST:CONTAINER[/tcp|udp]`; applies to `private` networking |
| `ps` | `-a, --all` | Show all containers instead of only running containers |
| `logs` | `-f, --follow` | Follow log output |


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
* Rootless private networking via `pasta`
* TCP/UDP host port publishing
* Loopback and resolver setup in isolated namespaces
* Automatic fallback from private networking to disabled networking when `pasta` is unavailable


## Far off goals (for now)

* Cgroups / resource limits
* Volume mounts
* More configuration options
* Security hardening
* Full OCI spec compliance

## Notes

In rootless mode, privilege-drop flows inside the container do not work reliably. Switching from container root to another user/group after startup with tools like `setpriv`, `su`, or similar mechanisms can fail because unprivileged GID mapping requires disabling `setgroups(2)`. Prefer the image's `USER` or `crate run --user ...` when the image can start directly as its final service user.
