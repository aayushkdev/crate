# Chapter 9 - Lifecycle, Signals, and Logs

Once a container process exists, a runtime still needs to manage it. That means tracking state, stopping it cleanly, and preserving output.

## The Problem

Image pulling, filesystem setup, and `exec` get a workload running, but they do not answer operational questions:

- is this container still alive?
- how do we stop it without leaving children behind?
- where does stdout go after detached start?
- how do later commands find the right process again?

Lifecycle is the layer that turns one process launch into a usable runtime, and later back into removable on-disk state.

## How Linux Does It

Linux gives you PIDs, signals, process groups, and ordinary files. A runtime builds lifecycle behavior from those primitives.

You can see the same ideas without containers:

```sh
sh -c 'trap "echo TERM; exit 0" TERM; while :; do sleep 1; done' &
pid=$!
kill -TERM "$pid"
wait "$pid"
```

Process groups matter when one workload starts children:

```sh
setsid sh -c 'sleep 100 & wait' &
pgid=$!
kill -TERM "-$pgid" # send the signal to the whole process group
```

## How Crate Uses It

Crate persists container lifecycle data in [`internal/container/state.go`](../internal/container/state.go):

```go
// internal/container/state.go
type State struct {
    ID         string    `json:"id"`
    Name       string    `json:"name"`
    Image      string    `json:"image"`
    Command    []string  `json:"command,omitempty"`
    Status     Status    `json:"status"`
    PID        int       `json:"pid,omitempty"`
    ExitCode   int       `json:"exit_code,omitempty"`
    LogPath    string    `json:"log_path,omitempty"`
    CreatedAt  time.Time `json:"created_at,omitempty"`
    StartedAt  time.Time `json:"started_at,omitempty"`
    FinishedAt time.Time `json:"finished_at,omitempty"`
}
```

Launch-time state and logging are recorded in [`internal/runtime/launch.go`](../internal/runtime/launch.go):

```go
// internal/runtime/launch.go
if !attach {
    cmd.SysProcAttr.Setpgid = true // detached container gets its own process group
    cmd.Stdout = logFile
    cmd.Stderr = logFile
}
```

After the child starts, Crate stores the running PID:

```go
// internal/runtime/launch.go
state := &container.State{
    ID:        containerID,
    Image:     cfg.Image,
    Command:   argv,
    Status:    container.StatusRunning,
    PID:       cmd.Process.Pid,
    LogPath:   logPath,
    StartedAt: time.Now().UTC(),
}
```

Stopping uses the standard TERM-then-KILL sequence in [`internal/runtime/stop.go`](../internal/runtime/stop.go):

```go
// internal/runtime/stop.go
if err := killProcessGroup(state.PID, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
    return err
}

if waitForExit(state.PID, 1*time.Second) {
    return container.UpdateState(containerID, func(s *container.State) {
        s.Status = container.StatusStopped
        s.FinishedAt = time.Now().UTC()
    })
}

if err := killProcessGroup(state.PID, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
    return err
}
```

Finally, read paths such as `ps` and `logs` use state refresh rather than a background daemon:

```go
// internal/container/state.go
if state.Status == StatusRunning && !ProcessAlive(state.PID) {
    state.Status = StatusExited
    if state.ExitCode == 0 {
        state.ExitCode = -1 // exited but exit status was not captured
    }
    if err := writeState(id, state); err != nil {
        return nil, err
    }
}
```

Removal follows the same file-based model:

```go
// internal/container/remove.go
state, err := RefreshState(id)
if err != nil {
    return err
}
id = state.ID // resolved from name if a name was given

if state.Status == StatusRunning || state.Status == StatusStopping {
    return fmt.Errorf("container %s is running; stop it first", id)
}

return removeContainerDir(id)
```

All lifecycle commands (`start`, `stop`, `logs`, `rm`, `ps`) resolve names to IDs internally via `resolveContainerID`. The function first scans every container's config for a matching `Name`, then falls back to hex-ID matching — so a name like `my-container` can be used anywhere an ID is accepted.

> Under the Hood
>
> Crate's lifecycle model is file-based, not daemon-based. That is a deliberate teaching tradeoff: state stays inspectable with normal tools.

> ⚠ Watch out
>
> Detached mode is not the same thing as a supervisor. Crate records state and returns control, but it does not introduce a long-lived management daemon.

## Connecting the Dots

This chapter closes the loop. The earlier chapters explained how Crate obtains image content, builds a rootfs, launches a namespaced process, gives that process terminal semantics when needed, and `exec`s the workload. Lifecycle is what makes all of that usable after startup.

## Try It Yourself

Run:

```sh
id=$(go run ./cmd/crate run -d alpine sh -c 'echo started; sleep 20')
go run ./cmd/crate ps -a
go run ./cmd/crate logs "$id"
go run ./cmd/crate stop "$id"
go run ./cmd/crate ps -a
```

Watch how `state.json` and the visible status evolve across those commands. Then stop and remove the container:

```sh
go run ./cmd/crate rm "$id"
go run ./cmd/crate container prune
```

## Key Takeaways

- Lifecycle management is built from ordinary Linux primitives: files, PIDs, signals, and process groups.
- Crate persists container state explicitly instead of depending on a daemon.
- `crate rm` and `crate container prune` are directory cleanup operations guarded by persisted state.
- TERM followed by KILL is a policy choice that balances graceful shutdown and predictability.
