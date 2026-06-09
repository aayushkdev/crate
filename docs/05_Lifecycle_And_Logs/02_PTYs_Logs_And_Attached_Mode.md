# PTYs, Logs And Attached Mode

Crate has two launch styles: attached and detached.

Attached containers need terminal behavior. Detached containers need durable logs.

## What A PTY Is

PTY stands for pseudo terminal.

A terminal program usually expects to talk to a terminal device, not just a plain file or pipe. A PTY gives us a pair of connected devices:

```text
terminal side used by Crate <-> terminal side seen by the child process
```

Crate writes your keyboard input into one side. The container process reads it from the other side. The container writes output to its side, and Crate relays that output back to your real terminal.

This is why a shell inside an attached container can do line editing, prompt drawing and terminal control.

Without a PTY, many programs still run, but they know they are not attached to a real terminal. They may disable colors, line editing, job control or full-screen behavior.

## Attached Mode

In attached mode, Crate starts the child behind a pseudo terminal. This is implemented in `internal/runtime/pty.go`.

The PTY path:

* starts the child with a PTY;
* puts the user's terminal into raw mode when possible;
* copies stdin to the PTY;
* copies PTY output to stdout;
* writes PTY output into the container log file;
* listens for `SIGWINCH` to handle terminal resize.

This is why an interactive shell behaves like a shell instead of a program connected to plain pipes.

## Detached Mode

Detached mode does not attach the user's terminal. Crate starts the child in its own process group and connects stdout/stderr to:

```text
containers/<id>/logs/container.log
```

The command prints the container ID and returns.

## logs

`crate logs` reads the container log file.

`crate logs -f` prints existing content, then polls for appended content while the container remains running.

This is intentionally simple. There are no log drivers yet, no structured logging, and no daemon-owned log stream.

## Why Pipes Are Not Enough

Many terminal programs behave differently when stdout is not a terminal. Shells may disable job control, full-screen programs may fail, and line editing can behave strangely.

A PTY gives the child a terminal-like device. From the program's point of view, it is attached to a real terminal even though Crate is relaying bytes between the user's terminal and the child.

This is why attached mode is more complex than simply setting `cmd.Stdout = os.Stdout`.

## Raw Mode

When possible, Crate puts the user's terminal into raw mode while relaying attached input.

Raw mode lets key presses pass through without the host terminal line discipline interpreting them first. This is important for interactive shells and programs that expect to handle control characters themselves.

Crate restores the terminal state afterward.

## Resize Handling

Terminal size matters for shells, pagers and full-screen programs.

Crate listens for `SIGWINCH`, the signal sent when the terminal window size changes. The PTY code can then propagate size changes so the program inside the container can redraw correctly.

This is a small detail, but it is part of making attached containers feel normal.

## Logs From Attached Containers

Attached output is also written to the container log file.

This means a user can run an attached container, exit it, and still inspect output later with:

```sh
crate logs <container>
```

The log file is not only for detached containers. Detached mode simply relies on it more heavily because there is no live terminal relay.

## Limitations

Crate does not yet support:

* attaching to an already-running detached container;
* multiple log drivers;
* structured log records;
* log rotation;
* timestamps in log output.

Those are Docker-level operational features rather than core container mechanics.

## Attached Mode And Exit

In attached mode, the parent stays connected to the PTY until the relay ends. Then it waits for the process and finalises container state.

If the relay fails, Crate kills the process group, waits, finalises state and returns the relay error. This avoids leaving a half-attached container running after the terminal side has broken.

## Detached Mode And Reaping

Detached mode returns control to the user quickly. Crate starts a goroutine to wait for the process and update state.

This is simple, but it depends on the launching command remaining alive long enough to reap. If it does not, later commands can still refresh stale state by checking the stored PID.

Again, this is the daemonless tradeoff. The implementation is easy to inspect, but it is not a full supervisor.

## Logs Are Byte Streams

Crate's log file is just the process output stream. It does not store metadata per line.

That means the log format is simple and compatible with ordinary tools:

```sh
tail -f ~/.local/share/crate/containers/<id>/logs/container.log
```

The downside is that Crate cannot currently filter logs by timestamp or stream structured records. Those features would require a richer log format or log driver system.

## Terminal State Cleanup

Any program that puts the user's terminal into raw mode has to be careful to restore it.

Crate defers terminal restoration after raw mode is enabled. If it did not, a failed or interrupted attached session could leave the host terminal in a strange state where input is not line-buffered or echo behaves unexpectedly.

This is not container-specific, but it is part of making a runtime pleasant to use.

## PTY Errors

PTY relays can end with `EIO` when the slave side closes. That can be normal during process exit.

The runtime has to distinguish expected PTY closure from real relay failures. This is another example where terminal behavior has Unix-specific edge cases that are hidden by a simple phrase like "attached mode".

## What attach Would Add

Crate can start a container attached, but it cannot yet attach to an already-running detached container.

Supporting that would require storing enough terminal and process information to connect a later CLI invocation to the running process. It may also require a long-lived shim or socket, which would challenge Crate's daemonless design.

That makes `attach` a more complex feature than it first appears.
