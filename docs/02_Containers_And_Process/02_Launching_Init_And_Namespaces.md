# Launching Init And Namespaces

Crate launches containers by starting a child copy of itself:

```text
/proc/self/exe init <container-id>
```

The internal `init` command is not a user command. It is the setup program that runs inside the new namespaces before becoming the real container command.

## Why Reuse /proc/self/exe

Using `/proc/self/exe` means Crate can re-enter the same binary without needing to find its installed path. The parent process starts the child with special process attributes, and the child follows the internal init path.

This keeps the launch mechanism self-contained.

## Namespace Flags

The parent configures `SysProcAttr.Cloneflags` in `internal/runtime/launch.go`.

Rootful containers use:

* `CLONE_NEWUTS` for hostname isolation;
* `CLONE_NEWPID` for a new PID tree;
* `CLONE_NEWNS` for mount isolation.

Rootless containers add:

* `CLONE_NEWUSER` for user namespace mapping.

Networking modes that require isolation add:

* `CLONE_NEWNET`.

## Synchronisation

Some setup must happen in the parent after the child exists. Rootless UID/GID mapping is the main example. Private networking setup also needs the child PID.

Crate creates a pipe and exposes it to the child through:

```text
CRATE_SYNC_FD
```

The child blocks until the parent writes a byte. This prevents the child from racing ahead before mappings or network setup are ready.

## Inside Init

The internal init path does the final setup:

1. Read container config.
2. Wait for parent setup when required.
3. Re-exec after rootless user mapping when required.
4. Apply runtime network mode override.
5. Open configured bind mounts.
6. Set hostname.
7. Set up the root filesystem.
8. Apply bind mounts.
9. Bring up loopback for isolated networking.
10. Wait for the private network interface if needed.
11. Apply working directory.
12. Apply container user.
13. Resolve command and executable path.
14. Call `execve`.

After `execve`, Crate's init code is gone. The same PID is now the container program.

## Why There Is An Internal init

Some setup must happen after namespace creation.

For example, mounting `/proc` before entering the PID namespace would give the wrong process view. Setting the hostname only makes sense inside the new UTS namespace. Resolving `/bin/sh` should happen after the root filesystem switch, because `/bin/sh` should be the image's shell, not the host's.

The internal init process is the place where those operations happen in the correct context.

This is similar in spirit to early OS development, where a boot path prepares the environment and then hands control to the next stage. Here, the parent process prepares the child, the child prepares the container environment, and then `execve` hands control to the actual program.

## PID Namespace Consequence

With `CLONE_NEWPID`, the first process inside the namespace becomes PID 1 from the container's point of view.

PID 1 has special behavior on Linux. Signal handling and child reaping can differ from ordinary processes. Crate does not yet implement a full init process or subreaper model. It simply execs the requested command as PID 1.

That is enough for simple containers, but it is a difference from more complete runtimes that may run an init shim or support better process reaping.

## Mount Namespace Consequence

`CLONE_NEWNS` gives the child a separate mount namespace. This is what lets Crate make `/` private, switch roots, mount `/proc`, mount `/dev`, and apply bind mounts without changing the host's mount table.

The mount namespace alone is not enough. Crate still has to mark propagation private, because mount namespaces can inherit shared propagation settings from the host. That is why filesystem setup starts with a recursive private mount.

## UTS Namespace Consequence

`CLONE_NEWUTS` gives the container its own hostname.

Crate derives the hostname from the container name and clamps it to Linux's hostname byte limit. This is a small example of a user-facing field becoming namespace state.

## Network Namespace Consequence

Crate only creates a network namespace for modes that need one:

* `none`;
* `private`.

Host networking intentionally does not use `CLONE_NEWNET`. The container shares the host network stack, so there is no separate interface setup and no port publishing boundary.

## Sync Pipe Failure Modes

The sync pipe prevents the child from continuing too early. If the parent fails while setting up mappings or networking, it closes the pipe and cleans up the child.

Without this synchronization, rootless init could try to mount, chroot or set users before UID/GID mappings exist. Private networking could also race with the interface setup.

The sync protocol is intentionally tiny: one inherited file descriptor and one byte. That keeps it easy to audit.

## What Happens If Launch Fails

Launch can fail at several points:

* resolving the entrypoint;
* resolving runtime networking;
* opening the log file;
* starting the child process;
* writing user namespace mappings;
* starting `pasta`;
* updating container state.

Crate tries to clean up partial launch state when these failures happen. For example, if network setup fails after the child exists, the runtime kills the process group, waits for the child and tears down network state.

This is important because container startup crosses a boundary from simple file preparation into live host processes. Once a process has been created, failure handling must be active cleanup, not just returning an error.

## Why State Is Updated After Start

The container is not marked `running` before the child exists. Crate starts the process first, then writes the running state with the child PID.

This ordering prevents state from claiming a container is running before there is a process to signal or inspect.

There is still a small window between process start and state write. A daemonless runtime cannot remove every race, but it can keep the sequence simple and recoverable.

## Comparing create And start

`crate create` stops before launch. It prepares the rootfs and writes config/state.

`crate start` reads that config and runs the launch path.

`crate run` combines both:

```text
create
start
cleanup if start fails
```

That makes `run` convenient for users while preserving separate implementation stages for readers.
