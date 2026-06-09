# Daemonless And Rootless Design

Two design choices shape most of Crate's implementation: it is daemonless, and it supports rootless execution.

## Daemonless

Docker has a long-running daemon. The CLI sends requests to that daemon, and the daemon owns most state.

Crate does not do this. The command you run performs the work itself and records durable state under:

```text
~/.local/share/crate
```

That means:

* image metadata is JSON in `images`;
* blobs are files in `blobs`;
* containers are directories in `containers`;
* container config and state are JSON files;
* logs are ordinary files.

This makes the runtime easier to study. There is no hidden server loop between the CLI and the Linux operations.

## Rootless

Rootless mode is used when Crate is run by a non-root user.

Crate creates a user namespace and maps container root to the current host user. It reads delegated ID ranges from `/etc/subuid` and `/etc/subgid`, writes `deny` to `/proc/<pid>/setgroups`, and uses `newuidmap` and `newgidmap` to install mappings.

Rootless mode changes other parts of the runtime too:

* filesystem switching uses `chroot` instead of `pivot_root`;
* `/sys` is skipped;
* private networking uses `pasta`;
* images that start as root and later call `setgroups(2)` may fail to drop privileges.

This is why rootless support is not just a boolean passed through the code. It affects process launch, mounts, users and networking.

## Why This Matters For Reading

When reading Crate, keep asking:

```text
is this state stored on disk because there is no daemon?
is this branch different because the runtime is rootless?
```

Those two questions explain many choices that otherwise look arbitrary.

## Daemonless Tradeoffs

The daemonless design makes Crate easy to inspect, but it also creates tradeoffs.

A daemon can supervise long-running containers continuously. It can keep an in-memory process table, subscribe to child exits, stream logs, manage restart policies and coordinate concurrent operations through one central service.

Crate does less. It writes state to disk and lets later commands refresh that state. If a detached container exits while no Crate command is watching, a later `crate ps` can notice the stale PID and update the status.

This is simpler and good for learning, but it is not the same as full supervision.

## Why State Files Are Still Useful

Plain files are enough for many operations:

```text
ps      -> read state and config
logs    -> read container.log
stop    -> read PID and send signals
rm      -> remove container directory
images  -> read image metadata
rmi     -> rewrite image metadata
prune   -> scan metadata and blobs
```

This gives Crate a very direct relationship between CLI commands and filesystem state. If a command behaves strangely, inspect the files first.

## Rootless Tradeoffs

Rootless support is valuable because it avoids requiring host root for the learning path. It also demonstrates how modern container runtimes use user namespaces.

But rootless mode has limits:

* device creation is restricted;
* mounting filesystems is restricted;
* GID mapping requires `setgroups` to be denied;
* private networking needs a helper like `pasta`;
* some images assume rootful privilege-drop behavior.

Crate should make these differences visible. Hiding them would make the code harder to learn from.

## How To Think About root Inside The Container

In rootless mode, root inside the container is not root on the host.

The mapping is closer to:

```text
container root is a namespaced identity backed by your host user
```

That is why rootless containers can create files that appear owned by your user or by delegated subuid/subgid ranges on the host.

This is also why bind mounts need care. If your host user can write a mounted path, the rootless container may be able to write it too. User namespaces reduce privilege, but they do not make every host path safe to expose.

## Reading With These Designs In Mind

When you see a JSON write, ask why the information has to survive after the command exits.

When you see a rootless branch, ask which Linux privilege rule forced that branch.

When you see a parent-child sync pipe, ask what setup would race if the child continued immediately.

Those questions make the code much easier to follow than reading function names alone.
