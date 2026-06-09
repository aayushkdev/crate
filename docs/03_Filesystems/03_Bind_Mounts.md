# Bind Mounts

Bind mounts let the user expose a host path inside the container:

```sh
crate run -v /host/path:/container/path[:ro] alpine sh
```

The parsing and validation live in `internal/container/mount.go`. The actual mount calls happen in `internal/runtime/mount.go`.

## Why Mounts Are Opened Early

Host paths are validated before launch. Crate also opens mount sources before the child fully enters the container filesystem.

This matters because after the root switch, the host path might not be reachable by the same pathname. Passing an opened file descriptor through the launch path keeps the source available.

## Applying The Mount

Inside init, Crate prepares the destination path. If the source is a directory, the destination must be a directory. If the source is a file, Crate creates the parent directory and file path as needed.

Then it calls a bind mount:

```text
mount(source, destination, "", MS_BIND|MS_REC, "")
```

For read-only mounts, it remounts:

```text
MS_BIND | MS_REMOUNT | MS_RDONLY | MS_REC
```

The two-step process is normal for read-only bind mounts.

## Current Limits

Crate supports simple bind mounts. It does not yet expose Docker's full mount model: named volumes, tmpfs mounts from the CLI, propagation modes, copy-up behavior or volume drivers.

## File Mounts Versus Directory Mounts

Bind mounts can target files or directories, and Crate has to preserve that distinction.

If the source is a directory, the destination should be a directory. If the source is a file, the destination should be a file. Creating the wrong type can hide image contents unexpectedly or make the mount fail.

This is why Crate stats the source and prepares the destination based on the source type.

## Mounts Happen After Root Setup

User bind mounts are applied after the root filesystem switch and after the standard container mounts.

That ordering means destinations are interpreted inside the container root. For example:

```sh
crate run -v /tmp/data:/data alpine sh
```

mounts the host path at:

```text
<container-rootfs>/data
```

from the container's point of view, it is simply:

```text
/data
```

## Read-Only Is A Remount

The read-only case is a good example of a Linux mount detail that abstractions hide.

Crate cannot make a bind mount read-only only by adding `MS_RDONLY` to the first bind call in the portable pattern it uses. It first creates the bind mount, then remounts the bind target as read-only.

That is why the implementation has two mount calls for `:ro` mounts.

## Mounts As Intentional Boundary Crossings

Bind mounts intentionally cross the container filesystem boundary.

That is their purpose. They let the host provide data, configuration, sockets or source code to a container. But this also means a bind mount can weaken isolation if it exposes sensitive host paths.

Rootless mode limits what the process can do as a host user, but it does not make every bind mount safe. If the current host user can read or write the mounted path, the rootless container can often do the same through its mapped identity.

Crate currently treats bind mounts as an explicit user request. A future security-focused mode could add warnings for dangerous paths.

## Examples

Mount a source directory:

```sh
crate run -v "$PWD":/src alpine ls /src
```

Mount it read-only:

```sh
crate run -v "$PWD":/src:ro alpine sh -c 'touch /src/new-file'
```

The second command should fail to create the file inside `/src`, because Crate remounts the bind target read-only.

Mount a file:

```sh
crate run -v /etc/hostname:/host-hostname:ro alpine cat /host-hostname
```

The destination is a file path, not a directory. Crate prepares the destination based on the source file type.

## What Named Volumes Would Add

Docker named volumes are not just bind mounts with shorter names. They introduce a runtime-managed storage object with its own lifecycle.

A future Crate volume implementation would need to answer:

* where named volumes live;
* how volume names are resolved;
* whether removing a container removes anonymous volumes;
* whether image data is copied into an empty volume on first use;
* how volume references are shown in inspect-like output.

For now, bind mounts are enough to show the Linux mount mechanism directly.
