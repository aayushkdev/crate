# Lifecycle And Logs

Crate's lifecycle model is built around files, PIDs and signals.

Because there is no daemon, lifecycle commands do not query a running service. They read JSON state, check process liveness, send signals, tail log files and remove directories.

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

