# Lifecycle And Logs

Crate's lifecycle model is built around files, PIDs and signals.

Because there is no daemon, lifecycle commands do not query a running service. They read JSON state, check process liveness, send signals, tail log files and remove directories.

## Beginner Terms

A signal is a small notification sent to a process. `SIGTERM` asks a process to exit. `SIGKILL` forces it to die. Crate uses these when stopping containers.

A process group is a set of related processes. Sending a signal to a process group can stop the main process and its children together.

A log is the recorded output of a container process. Crate writes container stdout and stderr to `container.log`.

A PTY, or pseudo terminal, is a terminal device pair used to make an interactive program think it is connected to a real terminal. Shells, editors and full-screen terminal programs usually need a PTY to behave correctly.

Attached mode connects your terminal to the container process through a PTY. Detached mode runs the process in the background and writes output to the log file.

State is Crate's recorded view of a container: created, running, exited, stopped or failed.

The main implementation files are:

* `internal/container/state.go`
* `internal/container/remove.go`
* `internal/container/prune.go`
* `internal/runtime/lifecycle.go`
* `internal/runtime/stop.go`
* `internal/runtime/logs.go`
* `internal/runtime/ps.go`
* `internal/runtime/pty.go`
* `internal/runtime/process.go`
* `internal/runtime/warnings.go`

## Chapters

* [State Transitions And ps](01_State_Transitions_And_PS.md)
* [PTYs, Logs And Attached Mode](02_PTYs_Logs_And_Attached_Mode.md)
* [Stop, Remove, Prune And Future Work](03_Stop_Remove_Prune_And_Future_Work.md)
