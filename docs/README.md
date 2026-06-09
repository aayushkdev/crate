# Crate Notes

Crate is a small daemonless container runtime written in Go for Linux.

These notes are written in the style of a project book: first explain the concept, then trace the idea into the implementation. The point is not only to say that containers use namespaces, mounts and images. The point is to show which concrete steps Crate performs when it turns an image name into a running Linux process.

Crate is useful to study because it keeps the main path visible:

```text
image name
    -> registry manifest
    -> local blobs and metadata
    -> container config and rootfs
    -> child process in namespaces
    -> filesystem and networking setup
    -> execve
    -> state, logs and cleanup
```

There is no long-running daemon in the middle. Commands read and write ordinary files under `~/.local/share/crate`, so most of the runtime can be understood by following the command that is currently running.

## Current Parts

* [Part 0: Introduction](00_Introduction/README.md)
    * [Building And Trying Crate](00_Introduction/01_Building_And_Trying_Crate.md)
    * [Daemonless And Rootless Design](00_Introduction/02_Daemonless_And_Rootless_Design.md)
    * [Reading The Codebase](00_Introduction/03_Reading_The_Codebase.md)
* [Part 1: Images And Storage](01_Images_And_Storage/README.md)
    * [Image References And Pulling](01_Images_And_Storage/01_Image_References_And_Pulling.md)
    * [Manifests, Blobs And Metadata](01_Images_And_Storage/02_Manifests_Blobs_And_Metadata.md)
    * [Layer Extraction And Image Pruning](01_Images_And_Storage/03_Layer_Extraction_And_Image_Pruning.md)
* [Part 2: Containers And Process](02_Containers_And_Process/README.md)
    * [Container Config, State, Names And IDs](02_Containers_And_Process/01_Config_State_Names_And_IDs.md)
    * [Launching Init And Namespaces](02_Containers_And_Process/02_Launching_Init_And_Namespaces.md)
    * [Rootless User Namespaces](02_Containers_And_Process/03_Rootless_User_Namespaces.md)
    * [Entrypoint, User And execve](02_Containers_And_Process/04_Entrypoint_User_And_Execve.md)
* [Part 3: Filesystems](03_Filesystems/README.md)
    * [Switching Root](03_Filesystems/01_Switching_Root.md)
    * [proc, sys, dev And run](03_Filesystems/02_Proc_Sys_Dev_And_Run.md)
    * [Bind Mounts](03_Filesystems/03_Bind_Mounts.md)
* [Part 4: Networking](04_Networking/README.md)
    * [Network Modes](04_Networking/01_Network_Modes.md)
    * [Rootless Private Networking With pasta](04_Networking/02_Rootless_Private_Networking_With_Pasta.md)
* [Part 5: Lifecycle And Logs](05_Lifecycle_And_Logs/README.md)
    * [State Transitions And ps](05_Lifecycle_And_Logs/01_State_Transitions_And_PS.md)
    * [PTYs, Logs And Attached Mode](05_Lifecycle_And_Logs/02_PTYs_Logs_And_Attached_Mode.md)
    * [Stop, Remove, Prune And Future Work](05_Lifecycle_And_Logs/03_Stop_Remove_Prune_And_Future_Work.md)
* [Extras: Appendices](99_Appendices/README.md)
    * [Command Reference](99_Appendices/A_Command_Reference.md)
    * [Storage Layout](99_Appendices/B_Storage_Layout.md)
    * [Task To Syscall Map](99_Appendices/C_Task_To_Syscall_Map.md)

## How To Read These Notes

Read the parts in order the first time. Later chapters assume the earlier flow:

1. Introduction explains how to build Crate and why daemonless/rootless design matters.
2. Images explain where root filesystem data comes from.
3. Containers explain how Crate records what should be started.
4. Filesystems explain how the process gets its view of `/`.
5. Networking explains how that process gets, or does not get, network access.
6. Lifecycle explains how Crate observes and cleans up the process later.
7. Appendices provide references that are useful while hacking on the project.

When a chapter mentions a source file, that file is the reference implementation for the concept being described.
