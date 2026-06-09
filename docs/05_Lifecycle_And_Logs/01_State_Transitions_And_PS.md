# State Transitions And ps

Container state lives in:

```text
containers/<id>/state.json
```

The known statuses are:

* `created`
* `running`
* `stopping`
* `exited`
* `stopped`
* `failed`

## Created

`container.Create` writes `created` after the rootfs, config and initial state are prepared.

At this point, no process exists. The container can be started later with `crate start`.

## Running

`launchContainer` writes `running` after the child process starts. The state records:

* host PID;
* command;
* log path;
* network mode;
* started time.

This is the state `crate ps` mostly displays.

## Exited

Attached containers are waited on directly. Detached containers are reaped by a goroutine in the command that launched them.

Since Crate is daemonless, the launching command may not always be around forever. Later commands call `RefreshState`, which checks whether the stored PID is still alive with `kill(pid, 0)`. If a container was marked running but the process is gone, Crate updates it to `exited`.

This is a simple recovery mechanism for stale state.

## ps

`crate ps` scans container directories, reads config and state, refreshes liveness, and prints summaries.

With no flags it shows running containers. With `-a`, it shows all containers.

The display includes image, command, status, PID, age, network mode and ports when available.

## State Is Best Effort

Because Crate is daemonless, state is necessarily best effort.

If the command that launched a detached container exits normally, it can reap the child and update state. If something interrupts that path, the state file might still say `running` even after the process is gone.

Crate handles this by refreshing state when later commands inspect it. This is not as strong as a daemon supervising every container, but it keeps the system understandable.

## Why kill(pid, 0)

`kill(pid, 0)` does not send a signal. It asks the kernel whether the process exists and whether the caller can signal it.

Crate uses this as a liveness check. If a state file says a container is running but the PID is no longer alive, Crate updates the state to `exited`.

This is another example of a low-level operation serving a higher-level lifecycle task.

## Created Versus Stopped

`created` means the container has never successfully run. The rootfs and config exist, but there is no recorded process execution.

`stopped` means Crate intentionally stopped a running process.

`exited` means the process ended. It may have exited normally, failed, or been detected as gone later.

These distinctions are useful for commands and for users. A future `inspect` command could expose them more completely.

## ps Without A Daemon

`crate ps` reconstructs its table every time:

1. Read container directories.
2. Read each state file.
3. Refresh process liveness.
4. Read config for network details.
5. Format rows.

There is no cached table. This makes the command slower than a daemon query for huge numbers of containers, but it is simple and robust for Crate's scope.

## Exit Codes

When a container process exits, Crate records an exit code when it can.

If a process is discovered as gone later through stale state refresh, Crate may not know the real exit code. In that case it records a sentinel value instead of inventing a successful exit.

This is another daemonless tradeoff. A supervising daemon can wait on children continuously and collect exact exit statuses. Crate can only do that while the launching command or waiting path is still alive.

## CreatedAt, StartedAt And FinishedAt

Crate records multiple timestamps because they answer different questions:

* `CreatedAt`: when the container directory and config were created;
* `StartedAt`: when a process was launched;
* `FinishedAt`: when Crate observed the process end.

For `crate ps`, the age display prefers the creation time when available. For debugging lifecycle behavior, the separate fields are more useful than one generic timestamp.

## Failed State

The `failed` status exists for launch or finalization errors.

Failures can happen after the container directory exists. For example, the child might start but setup could fail during mounts, networking or `execve`. Crate needs a way to record that the container did not become a normal running or exited process.

This is another reason creation and launch are separate stages.

## Reading State By Hand

Because state is JSON, it is worth opening it directly:

```sh
cat ~/.local/share/crate/containers/<id>/state.json
```

Compare that with:

```sh
crate ps -a
```

The `ps` command is mostly a formatted and refreshed view of these files.

This is a good debugging habit for Crate. If a lifecycle command behaves unexpectedly, inspect the state file and then compare it to process reality with:

```sh
ps -p <pid>
```

## Why There Is No Global Container Table

Crate does not keep a single `containers.json` file.

Each container owns its own state. This avoids one global file becoming a coordination point for every lifecycle command. It also means a damaged container state file does not necessarily make every other container unreadable.

The tradeoff is that list operations scan directories.

For a small daemonless runtime, this is a reasonable tradeoff. For thousands of containers, a daemon or database-backed index would be more efficient.
