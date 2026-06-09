# Container Config, State, Names And IDs

Container creation starts in `internal/container/create.go`.

It does work that can be done before any isolated process exists:

1. Parse the image reference.
2. Pull the image if metadata is missing.
3. Read image metadata.
4. Resolve or generate a container name.
5. Generate a container ID.
6. Build the container config.
7. Create the rootfs directory.
8. Apply image layers.
9. Write `config.json`.
10. Write initial `state.json`.

## Config

The config is the intended container setup. It records:

* ID and name;
* image;
* whether it is rootless;
* command, entrypoint, environment and working directory;
* user;
* network mode and published ports;
* auto-remove setting;
* bind mounts.

Image defaults are read from the image config blob. CLI options override some of those defaults. For example, `--user` replaces the image user, and `--env` merges with image environment variables.

## State

The state is the observed runtime status. It records:

* status;
* PID;
* exit code;
* command;
* log path;
* network mode;
* created, started and finished times;
* error text when needed.

New containers start as `created`. Launch updates them to `running`.

## Names

Crate supports optional names, but it does not maintain a name database. Name resolution scans existing container config files. If no name matches, Crate falls back to hex ID prefix matching.

This is a daemonless design pattern repeated across the project: use the filesystem as the source of truth.

## IDs

Crate generates IDs from six random bytes encoded as hex. These IDs are short enough to use comfortably and long enough for a small learning runtime.

The generated ID becomes the container directory name:

```text
~/.local/share/crate/containers/<id>
```

## Config Is A Snapshot

The container config is not a live view of the image. It is a snapshot of the settings Crate selected when the container was created.

This matters if an image tag moves later. Suppose we create a container from `alpine:latest`, then pull a newer Alpine. The existing container config still has the command, environment, user, working directory, mounts and networking chosen at creation time. Crate does not rebuild old containers just because image metadata changes.

This matches the way containers are usually understood: an image is used to create a container, but the container then has its own identity and state.

## Environment Merging

Image config can contain environment variables. The user can add or override variables with `--env`.

Crate merges those values before writing the container config. This means the init path does not need to know which variables came from the image and which came from the CLI. It receives one final environment list.

This pattern appears throughout Crate: resolve choices early, then keep later stages simple.

## Network Config In The Container Config

Network mode is selected during creation and saved in config.

Rootless containers default to private networking. Rootful containers default to host networking. User flags can override this with:

```sh
--network host
--network none
--network private
```

Port publishing is also stored in the config, but it only has meaning for private networking. If publishing is requested for a mode that cannot support it, Crate drops the ports and returns a warning.

Launch may still adjust the runtime config. For example, if rootless private networking was selected but `pasta` is missing, launch falls back to `none`.

## State Is Allowed To Change

Config should be stable. State changes.

The state file starts as:

```text
created
```

then can move through:

```text
running -> exited
running -> stopping -> stopped
running -> failed
```

The exact status names are in `internal/container/state.go`.

Because state is only a file, Crate has to refresh it. A process might exit while no Crate command is watching. `RefreshState` checks whether a stored running PID is alive and updates stale state when necessary.

## Why Names Are Scanned

Using a name database would require another file or service to keep in sync. Crate avoids that by scanning container config files.

This is slower than a database lookup, but the runtime is small and local. The tradeoff is worth it because it keeps the source of truth obvious:

```text
if a container directory exists and has config/state, Crate can reason about it
```

There is also a failure-mode benefit. If a command is interrupted, there is no separate name index that might point to a missing container or miss an existing one.
