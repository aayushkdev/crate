# Crate Book

This directory is a technical book about Crate, a container runtime written in Go.

It is intentionally organized as a sequence of lessons rather than a reference manual. Each chapter starts from the Linux primitive itself, then maps that primitive onto the real Crate implementation.

This edition covers the concepts Crate actually implements today:

1. [Image References and Pulling](./01-image-references-and-pulling.md)
2. [Manifests and Platform Selection](./02-manifests-and-platform-selection.md)
3. [Layer Tars and Union Filesystems](./03-layer-tars-and-union-filesystems.md)
4. [Container Creation and State](./04-container-creation-and-state.md)
5. [Namespaces and Rootless Launch](./05-namespaces-and-rootless-launch.md)
6. [Root Filesystem Switching and Mounts](./06-root-filesystem-switching-and-mounts.md)
7. [Entrypoint, PATH Lookup, and exec](./07-entrypoint-path-lookup-and-exec.md)
8. [Terminals and PTYs](./08-terminals-and-ptys.md)
9. [Lifecycle, Signals, and Logs](./09-lifecycle-signals-and-logs.md)
10. [Rootless Networking with pasta](./10-rootless-networking-with-pasta.md)
