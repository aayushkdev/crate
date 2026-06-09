# proc, sys, dev And run

After switching roots, Crate mounts a minimal set of filesystems that normal Linux programs expect.

The high-level order is in `internal/fs/setup.go`:

1. Open host device file descriptors.
2. Switch root.
3. Mount `/proc`.
4. Mount `/sys`.
5. Mount `/dev`.
6. Mount `/run`.

## /proc

Crate mounts:

```text
proc -> /proc
```

with restrictive flags. This gives programs a process view that matches the PID namespace.

Without `/proc`, many tools still start, but process-aware programs quickly become confused or lose useful functionality.

## /sys

Rootful containers get `/sys` mounted read-only with `nosuid`, `nodev` and `noexec`.

Rootless containers skip `/sys`. This is a practical limitation of the reduced privilege environment.

## /dev

Crate mounts `/dev` as `tmpfs` and then creates or binds a minimal device set:

* `/dev/null`
* `/dev/zero`
* `/dev/random`
* `/dev/urandom`
* `/dev/full`
* `/dev/shm`
* `/dev/pts`
* `/dev/ptmx`

Rootless mode cannot freely create device nodes, so Crate opens selected host device file descriptors before switching roots and then binds them into the container `/dev`.

This is why `OpenHostDevFDs` happens before `setupRootfs`.

## /run

`/run` is mounted as tmpfs. It gives programs a writable runtime directory that is not part of the image layer data.

## Why These Mounts Are Not In The Image

The image provides ordinary filesystem contents. It should not contain a live `/proc` or a live `/dev`.

Those filesystems represent runtime state:

* `/proc` represents processes and kernel information;
* `/sys` represents kernel device and subsystem information;
* `/dev` represents device nodes and pseudo devices;
* `/run` represents temporary runtime files.

Mounting them during container startup ensures they reflect the current container environment rather than stale files from an image layer.

## proc And PID Namespaces

`/proc` becomes especially important with PID namespaces.

If `/proc` is mounted after entering the PID namespace, process listings inside the container match the namespace. If the container reused the host's `/proc`, commands like `ps` would expose host processes and break isolation expectations.

This is why filesystem setup happens inside the internal init path after namespace creation.

## devpts And PTYs

Interactive containers rely on pseudo terminals. `/dev/pts` is where slave PTY devices appear.

Crate's `/dev` setup includes a `devpts` mount and `/dev/ptmx` handling so terminal programs can work. This connects the filesystem chapter with the lifecycle chapter: attached mode uses a PTY, and the container filesystem needs the standard PTY paths.

## Minimal Device Philosophy

Crate does not expose the host's entire `/dev`.

Instead, it creates or binds a small set of common devices. This is safer and easier to reason about. A production runtime would make this configurable and integrate with device cgroup rules, but the learning version keeps the list explicit.

The minimal set is enough for many userland programs:

* write output to `/dev/null`;
* read zeros or random data;
* use shared memory under `/dev/shm`;
* allocate pseudo terminals.

## Rootless Device Setup

Device nodes are privileged. In rootless mode, creating them with `mknod` is restricted.

Crate handles this by opening selected host device files before the root switch and then using bind mounts where possible. This explains why host file descriptors appear in filesystem setup: they are a way to carry access across the root boundary.

## Why The Order Matters

The order in `fs.Setup` is not arbitrary.

Crate opens host device file descriptors first because those paths may disappear from view after the root switch. It then switches root before mounting `/proc`, `/sys`, `/dev` and `/run`, because those mounts should appear inside the container filesystem.

If `/proc` were mounted before entering the container's mount namespace and root, it would not represent the container environment correctly. If `/dev` were prepared before the root switch, it would affect the wrong tree.

The setup order is therefore part of the isolation boundary.

## Runtime Filesystems Versus Image Files

It is useful to separate two categories:

```text
image files       -> unpacked from layers
runtime filesystems -> mounted during startup
```

Image files include `/bin/sh`, `/etc/passwd`, libraries and application files.

Runtime filesystems include `/proc`, `/dev`, `/run` and sometimes `/sys`.

This distinction explains why container creation and container launch are separate. Creation can unpack image files. Launch has to mount runtime filesystems because they depend on namespaces and the current process context.

## Production Gaps

Crate's filesystem setup is intentionally small. A production runtime would need more policy and compatibility work:

* masked paths;
* read-only paths;
* tmpfs options;
* device cgroup integration;
* mount propagation options;
* SELinux or AppArmor labels;
* OCI mount specification handling.

Crate focuses on the core mechanics first.
