# Chapter 7 - Entrypoint, PATH Lookup, and exec

At this point Crate has a process in new namespaces, with a new root filesystem and the expected support mounts. One question remains: what exact program should become PID 1?

## The Problem

Container startup cannot stop at "run the image". It must compute one final argument vector from:

- image `Entrypoint`
- image `Cmd`
- user-supplied command override
- environment-driven PATH lookup

Then it must replace the runtime process image with that target program.

## How Linux Does It

The kernel primitive is `execve(2)`.

Minimal example:

```c
// execve_demo.c
#include <unistd.h>

int main(void) {
    char *argv[] = {"/bin/sh", "-c", "echo hello from execve", NULL};
    char *envp[] = {"PATH=/usr/bin:/bin", NULL};
    execve(argv[0], argv, envp);
    return 1; // only reached if execve fails
}
```

Shells normally perform PATH lookup before `execve`, but a runtime can do that itself:

```sh
PATH=/usr/bin:/bin command -v sh
```

That distinction matters because Crate does not insert a shell layer automatically.

## How Crate Uses It

Crate resolves command precedence in [`internal/container/exec.go`](/home/aayush/projects/crate/internal/container/exec.go):

```go
// internal/container/exec.go
switch {
case len(cfg.EntryPoint) > 0 && len(userCmd) > 0:
    return append(cfg.EntryPoint, userCmd...), nil
case len(cfg.EntryPoint) > 0 && len(cfg.Cmd) > 0:
    return append(cfg.EntryPoint, cfg.Cmd...), nil
case len(cfg.EntryPoint) > 0:
    return cfg.EntryPoint, nil
case len(userCmd) > 0:
    return userCmd, nil
case len(cfg.Cmd) > 0:
    return cfg.Cmd, nil
default:
    return nil, fmt.Errorf("no command specified")
}
```

Then it resolves the executable path:

```go
// internal/container/exec.go
path := "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
for _, e := range env {
    if strings.HasPrefix(e, "PATH=") {
        path = strings.TrimPrefix(e, "PATH=")
        break
    }
}

for _, dir := range strings.Split(path, ":") {
    full := filepath.Join(dir, cmd)
    if st, err := os.Stat(full); err == nil && !st.IsDir() {
        return full, nil
    }
}
```

The final handoff is the irreversible step in [`internal/container/container.go`](/home/aayush/projects/crate/internal/container/container.go):

```go
// internal/container/container.go
cmd, err := ResolveEntrypoint(cfg, command)
Fatal(err)

execPath, err := resolvePath(cmd[0], env)
Fatal(err)

Fatal(syscall.Exec(execPath, cmd, env)) // runtime disappears here
```

This is the equivalent of the earlier raw Linux `execve` example, but with image-config-aware command composition and explicit PATH lookup.

> Under the Hood
>
> Once `syscall.Exec` succeeds, there is no Crate process left inside the container. The workload is now PID 1 in that namespace.

> ⚠ Watch out
>
> "command not found" here is a rootfs or PATH problem, not a shell problem. Crate does not invoke `/bin/sh -c` unless you ask it to.

## Connecting the Dots

The previous chapter built an isolated execution environment. This chapter places a real workload into that environment. The next chapter steps back out and looks at how Crate keeps managing that workload afterward through lifecycle state, logs, and signals.

## Try It Yourself

Compare these two commands:

```sh
go run ./cmd/crate run alpine /bin/sh -c 'echo hello'
go run ./cmd/crate run alpine sh
```

Then inspect [`internal/container/exec.go`](/home/aayush/projects/crate/internal/container/exec.go) and reason about why the second command depends on PATH and the first one does not.

## Key Takeaways

- Container startup ends with one concrete `argv`, not a vague notion of "run the image".
- Crate resolves `Entrypoint`, `Cmd`, and user overrides explicitly.
- PATH lookup is done by the runtime, not by an implicit shell wrapper.
- `syscall.Exec` is the handoff where the workload becomes PID 1.
