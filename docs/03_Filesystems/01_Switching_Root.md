# Switching Root

The root filesystem switch is the operation that makes `/bin/sh` inside the container refer to the image's `/bin/sh`, not the host's.

Crate prepares:

```text
~/.local/share/crate/containers/<id>/rootfs
```

then the internal init process moves into it.

## Private Mount Propagation

The first step is:

```text
mount("", "/", "", MS_PRIVATE|MS_REC, "")
```

This prevents mount events in the container mount namespace from propagating back to the host. Without this, mount work done for the container could leak outward depending on the host's propagation setup.

## Making rootfs A Mount Point

Crate bind mounts the rootfs onto itself:

```text
mount(rootfs, rootfs, "", MS_BIND|MS_REC, "")
```

`pivot_root` requires the new root to be a mount point. This self-bind is a common way to satisfy that requirement.

## Rootful pivot_root

In rootful mode, Crate uses `pivot_root`.

The sequence is:

1. Create `rootfs/.oldroot`.
2. Call `pivot_root(rootfs, rootfs/.oldroot)`.
3. Detach `/.oldroot`.
4. Remove `/.oldroot`.
5. `chdir("/")`.

This moves the old root out of the way and makes the image rootfs the process root.

## Rootless chroot

In rootless mode, Crate uses `chroot`.

This is weaker than `pivot_root`, but it fits the privileges available to an unprivileged user namespace path. The code still marks mounts private and bind mounts rootfs first, so the setup remains close to the rootful path.

This difference is one of the places where rootless support is visible in the implementation rather than just a flag.

## Why chroot Alone Is Not A Container

It is tempting to think of containers as "just chroot". Crate's filesystem path shows why that is incomplete.

`chroot` changes how absolute paths are resolved for a process. It does not create a PID namespace, does not mount `/proc`, does not create `/dev`, does not isolate networking, and does not prevent all privileged escape routes by itself.

In Crate, the root switch is only one step in a larger sequence:

```text
new namespaces
private mount propagation
rootfs bind mount
pivot_root or chroot
standard mounts
bind mounts
user switch
execve
```

The root filesystem is necessary, but it is not the whole container.

## Why The Old Root Must Go Away

In the rootful `pivot_root` path, the old root is temporarily visible at:

```text
/.oldroot
```

Crate immediately detaches and removes it. Leaving the old root mounted inside the container would be a serious isolation bug. A process could potentially access host files through that path.

The sequence:

```text
pivot_root
umount /.oldroot
remove /.oldroot
```

is therefore part of the isolation story, not just cleanup.

## Mount Propagation Example

On many Linux systems, parts of the mount tree can be shared. Shared propagation means mount events under one peer can propagate to another.

If a runtime enters a mount namespace but does not make propagation private, mounting something inside the container may still be visible elsewhere depending on the inherited propagation settings.

That is why Crate starts with:

```text
MS_PRIVATE | MS_REC
```

This recursively makes the namespace's mount tree private before adding container mounts.

## Rootless Tradeoff

Using `chroot` in rootless mode is a tradeoff. It allows the learning runtime to run as an ordinary user, but it is not equivalent to the rootful `pivot_root` setup.

This is a recurring theme in rootless containers: many operations are possible, but they require a slightly different implementation and sometimes have weaker semantics or extra restrictions.

Crate documents this difference rather than hiding it behind the word "rootless".

## Debugging The Root Switch

When debugging filesystem setup, it helps to remember which paths are host paths and which paths are container paths.

Before the root switch:

```text
rootfs = ~/.local/share/crate/containers/<id>/rootfs
```

After the root switch, that same directory is seen by the process as:

```text
/
```

So a destination like `/proc` means:

```text
~/.local/share/crate/containers/<id>/rootfs/proc
```

from the host's point of view, but simply:

```text
/proc
```

inside the container.

Many mount bugs are easier to understand once this perspective shift is clear.

## What pivot_root Protects Against

`pivot_root` does not only change path lookup. It also lets Crate detach the old root mount from the container's mount namespace.

That is stronger than leaving the old root reachable somewhere and relying on path discipline. The container process should not have a convenient route back to host `/`.

Rootless `chroot` cannot provide the exact same property in Crate's current implementation. That is one of the reasons rootless and rootful behavior are documented separately.
