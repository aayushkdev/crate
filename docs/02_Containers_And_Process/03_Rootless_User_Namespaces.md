# Rootless User Namespaces

Rootless mode is used when Crate is run by a non-root user.

The goal is to let the container process act like root inside its own namespace while remaining unprivileged on the host. This is one of the most important differences between a toy chroot wrapper and a real container runtime.

The implementation is in `internal/runtime/userns.go`.

## Mapping Container Root

Crate maps:

```text
container uid 0 -> current host uid
container gid 0 -> current host gid
```

Additional IDs are mapped from delegated ranges in:

```text
/etc/subuid
/etc/subgid
```

Crate accepts entries keyed by username or numeric UID.

## newuidmap and newgidmap

The parent process runs:

```text
newuidmap <pid> ...
newgidmap <pid> ...
```

These helper programs are required for multi-ID rootless mapping. If they are missing, rootless startup fails.

The mapping arguments are built as triples:

```text
container-id host-id size
```

The first range maps container root to the current user. Later ranges come from the subuid/subgid files.

## setgroups

Before writing the GID map, Crate writes:

```text
deny
```

to:

```text
/proc/<pid>/setgroups
```

Linux requires this for unprivileged GID mappings.

This has a practical side effect. Some image entrypoints start as root and later try to drop privileges using `setgroups(2)`. In rootless mode, that can fail. Crate warns when a rootless container starts as root and suggests `--user` if the image can start directly as its final user.

## Re-exec

After the parent writes the mappings, the child re-executes the internal init path with:

```text
CRATE_INIT_MAPPED=1
```

This gives the rest of init a clean execution after the namespace mappings are installed.

## Why Rootless Is More Than chroot

A plain `chroot` changes pathname lookup, but it does not make a process safe or isolated by itself. A process with host root privileges can often escape or affect the host in ways an ordinary user cannot.

Rootless containers approach the problem differently. The process may appear to be root inside the user namespace, but that root is mapped to an unprivileged host identity. The kernel checks permissions against the mapped host IDs when the process touches host resources.

This is why user namespaces are central to rootless mode. They are not just a convenience for avoiding `sudo`; they change what "root" means.

## Delegated ID Ranges

The current user's own UID/GID can map container root, but a container often needs more IDs than just 0. Files may be owned by many users. Processes may switch users. Packages may expect system users.

Linux distributions commonly delegate ranges in:

```text
/etc/subuid
/etc/subgid
```

An entry might look like:

```text
aayush:100000:65536
```

This says the user can map a range of host IDs starting at 100000. Crate reads these ranges and maps them after container ID 0.

## Why Helpers Are Needed

Writing UID/GID maps directly is restricted. The `newuidmap` and `newgidmap` helpers are installed setuid-root on many systems and enforce the delegation rules from `/etc/subuid` and `/etc/subgid`.

Crate does not try to bypass that policy. It shells out to the helpers and reports their errors.

This is an important practical detail. If rootless startup fails, the problem is often not in namespace creation itself, but in host configuration:

* missing helper binaries;
* no delegated subuid range;
* no delegated subgid range;
* helper not installed with the expected permissions.

## Why setgroups Breaks Some Images

The `setgroups` denial is required for safety, but it changes what programs inside the container can do.

Some entrypoint scripts start as root, create users or groups, then drop privileges. If they call a tool that needs `setgroups(2)`, rootless mode can fail even though rootful mode works.

Crate's warning tries to point at the practical workaround:

```sh
crate run --user <uid>:<gid> image
```

This only works when the image can start directly as that user. Some images require their root entrypoint to prepare files first, and those images may need runtime features Crate does not yet provide.

## Rootless Filesystem Differences

Rootless mode also affects the filesystem setup:

* Crate uses `chroot` rather than `pivot_root`;
* `/sys` is skipped;
* device setup is more limited;
* host device file descriptors are opened before the root switch.

This is why rootless mode appears in multiple packages. It is a cross-cutting runtime constraint, not only a launch flag.
