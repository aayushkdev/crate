# Chapter 8 - Terminals and PTYs

Choosing the right `argv` and `exec`ing it is not enough for interactive programs. A shell connected to plain pipes is still not a terminal.

## The Problem

Before PTY support, attached mode forwarded bytes directly:

- `stdin` came from the host terminal
- `stdout` and `stderr` went back to the host terminal and log file
- but the workload did not own a controlling terminal

That gap matters immediately for shells and terminal programs. Readline, job control, full-screen interfaces, and terminal resize handling all depend on terminal semantics, not only on readable and writable file descriptors.

The before/after is concrete:

- before: `sh` can read bytes, but behaves like a process behind pipes
- after: `sh`, `vi`, `top`, and similar programs see a real terminal

## How Linux Does It

Linux solves this with pseudo-terminals.

A PTY is a pair:

- a master side, which the runtime holds
- a slave side, which the child process sees as its terminal

You can inspect the same mechanism without Crate:

```sh
script -q -c /bin/sh /dev/null
```

That command allocates a PTY, runs a shell on the slave side, and forwards bytes through the master side. Inside the shell, you can confirm that stdin is a terminal:

```sh
tty
stty -a
```

A PTY also has a window size and a controlling-terminal relationship. In raw Linux terms, the runtime needs to:

- create a new session with `setsid(2)`
- make the slave side the controlling terminal
- relay bytes between the host terminal and the PTY master
- propagate `SIGWINCH` size changes

Minimal C sketch:

```c
// pty_demo.c
#define _XOPEN_SOURCE 600
#include <fcntl.h>
#include <stdlib.h>
#include <unistd.h>

int main(void) {
    int master = posix_openpt(O_RDWR | O_NOCTTY);
    grantpt(master);
    unlockpt(master);

    char *slave_name = ptsname(master);
    int slave = open(slave_name, O_RDWR);

    if (fork() == 0) {
        setsid();                    // child becomes session leader
        dup2(slave, 0);
        dup2(slave, 1);
        dup2(slave, 2);
        execl("/bin/sh", "sh", NULL);
        _exit(1);
    }

    write(master, "echo hello from pty\n", 20);
    pause();
}
```

That is the raw Linux equivalent of what a PTY library packages up for you.

## How Crate Uses It

Crate originally attached the foreground path with plain stdio forwarding:

```go
// internal/runtime/launch.go
if attach {
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
}
```

The raw Linux equivalent is just three inherited file descriptors and no terminal setup.

That path is enough for non-interactive output, but it does not create a controlling terminal for the container workload. The PTY path closes that gap.

Crate starts the attached process through a PTY in [`internal/runtime/pty.go`](../internal/runtime/pty.go):

```go
// internal/runtime/pty.go
func startAttached(cmd *exec.Cmd) (*os.File, error) {
    attrs := cmd.SysProcAttr
    attrs.Setsid = true   // child becomes a session leader
    attrs.Setctty = true  // slave side becomes the controlling terminal

    ptmx, err := pty.StartWithAttrs(cmd, nil, attrs)
    if err != nil {
        return nil, err
    }

    return ptmx, nil
}
```

The raw Linux equivalent is `setsid`, opening a PTY slave, and making that slave the controlling terminal before `exec`.

The host terminal side also needs to stop line-buffering and local echo:

```go
// internal/runtime/pty.go
if term.IsTerminal(int(os.Stdin.Fd())) {
    oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
    if err == nil {
        defer term.Restore(int(os.Stdin.Fd()), oldState)
    }
}
```

The raw Linux equivalent is changing the host terminal's termios flags before relaying bytes and restoring them afterward.

PTYs also need resize propagation:

```go
// internal/runtime/pty.go
resize := make(chan os.Signal, 1)
signal.Notify(resize, syscall.SIGWINCH)
defer func() {
    signal.Stop(resize)
    close(resize)
}()

go func() {
    for range resize {
        if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
            log.Printf("error resizing pty: %v", err)
        }
    }
}()

if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
    log.Printf("error resizing pty: %v", err)
}
```

The raw Linux equivalent is reading the host terminal window size with `ioctl(TIOCGWINSZ)` and applying it to the PTY slave whenever `SIGWINCH` arrives.

Finally, Crate relays bytes through the PTY master and treats PTY hangup correctly:

```go
// internal/runtime/pty.go
go io.Copy(ptmx, os.Stdin)
_, err := io.Copy(io.MultiWriter(os.Stdout, logFile), ptmx)
if err != nil && err != io.EOF && !errors.Is(err, syscall.EIO) {
    return err
}
```

The raw Linux equivalent is a bidirectional relay loop between the host terminal and the PTY master, with Linux PTY hangup treated as a normal end-of-session condition.

> Under the Hood
>
> A PTY is not just "nicer stdin/stdout". It changes what the program believes it is attached to. That affects line editing, signal generation, terminal modes, and job control, which is why an interactive shell behind pipes feels wrong even when byte transport itself works.

> ⚠ Watch out
>
> On Linux, reading from a PTY master after the slave side closes often returns `EIO`, not `EOF`.

## Connecting the Dots

The `exec` chapter explained how Crate chooses the final program and replaces itself with it. This chapter explains why attached interactive programs behave like terminal sessions instead of pipe-fed subprocesses. The next chapter can stay focused on lifecycle state, logs, signals, and removal because terminal semantics have already been separated out.

## Try It Yourself

Run these two commands and compare the behavior:

```sh
go run ./cmd/crate run alpine sh
go run ./cmd/crate run -d alpine sh -c 'tty; sleep 5'
```

In the attached case, type `tty`, resize your terminal, and notice that the shell behaves like a normal terminal session. In the detached case, inspect the output with:

```sh
go run ./cmd/crate logs <CONTAINER_ID>
```

The attached path has a PTY; the detached path does not.

## Key Takeaways

- Interactive shells need terminal semantics, not only stdin/stdout pipes.
- Crate attaches foreground workloads through a PTY master/slave pair and a controlling-terminal session.
- Host-side raw mode and `SIGWINCH` propagation are part of terminal correctness, not optional polish.
- PTY hangup on Linux commonly appears as `EIO` on the master side.
