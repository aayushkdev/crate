# Manifests, Blobs And Metadata

The manifest is the bridge between the registry and the local image store.

A single image manifest points to:

* one config blob;
* an ordered list of layer blobs.

An image index points to platform-specific manifests. Crate currently looks for Linux amd64 and then fetches that concrete manifest by digest.

The resolution code is in `internal/image/manifest.go`.

## Config Blob

The config blob is not a filesystem layer. It describes defaults for the container:

* environment variables;
* entrypoint;
* command;
* working directory;
* user;
* OS and architecture.

Crate reads this later in `internal/container/config.go` when building the container config.

## Layer Blobs

Layer blobs are compressed tar archives. They contain filesystem changes. Crate stores them by digest and later applies them in order into a fresh rootfs directory.

The order matters. Later layers can replace or delete files from earlier layers.

## Local Blob Store

Blob paths are defined in `internal/storage/paths.go`. A digest like:

```text
sha256:abc123
```

becomes:

```text
~/.local/share/crate/blobs/sha256/abc123
```

Both config blobs and layer blobs use this layout.

## Metadata Store

Metadata lives under:

```text
~/.local/share/crate/images
```

Each metadata file is keyed by manifest digest and records:

* short image ID, derived from the config digest;
* repository;
* repo tags;
* manifest digest;
* config digest;
* layer digests;
* OS and architecture;
* total layer size;
* local creation time.

The metadata store is what makes `crate images`, `crate rmi`, and `crate image prune` possible without a daemon.

## Atomic JSON Writes

`internal/storage/store.go` writes JSON through a temporary file and then renames it into place. This is a small but important daemonless-runtime detail. If a command is interrupted in the middle of a write, the final path is less likely to contain a half-written JSON document.

It is not a full database, but it is enough for Crate's simple state model.

## How The Manifest Becomes Local State

The registry manifest is a remote description. Crate's metadata file is the local description.

The two are related, but they are not the same thing. The registry manifest tells Crate:

```text
this config digest and these layer digests make up the image
```

The local metadata also records:

```text
which local tags point here
what short image ID to display
what platform Crate selected
how large the layers are
when Crate created this local record
```

That extra local information is what makes CLI commands pleasant. `crate images` should not have to contact the registry to display a table. It can read local metadata.

## Manifest Lists

Many modern images do not point directly at a single filesystem. They point at a manifest list or OCI index.

An index is roughly:

```text
linux/amd64   -> sha256:...
linux/arm64   -> sha256:...
linux/arm/v7  -> sha256:...
```

Crate currently searches for:

```text
linux/amd64
```

and fetches that manifest by digest.

This is intentionally simple, but it has consequences. On an ARM machine, Crate would still try to select Linux amd64 unless the implementation is extended. A production runtime would select based on the host platform or a user-provided `--platform`.

The useful lesson is that "an image tag" can name a platform index, not just one image. A runtime has to collapse that into a concrete manifest before it can unpack layers.

## Why The Config Digest Becomes The Display ID

Crate derives the short image ID from the config digest.

Docker-like tools often display image IDs based on image config identity rather than the manifest digest. The config represents the image's runtime defaults and rootfs data, while the manifest is the distribution wrapper that points to compressed blobs.

Crate's implementation is simpler than Docker's, but the idea is similar enough for a learning runtime:

```text
display a short stable ID derived from image content
```

The full content identity is still the manifest digest in local metadata.

## Shared Layers

Layer sharing is the reason the blob store is digest-addressed.

Suppose two tags use the same base layer:

```text
image A -> layer X, layer Y
image B -> layer X, layer Z
```

There should only be one local copy of layer X. When Crate downloads a blob, it checks if the digest already exists. When Crate prunes blobs, it checks all remaining metadata records before deleting anything.

This is a small version of the content-addressed storage model used by real container engines.

## Reading The Files Manually

Because the store is plain JSON and files, it is useful to inspect it by hand:

```sh
find ~/.local/share/crate/images -maxdepth 1 -type f
find ~/.local/share/crate/blobs -type f | head
```

Then open one metadata file and compare:

* `repoTags`;
* `manifestDigest`;
* `configDigest`;
* `layers`.

Those fields are enough to understand why a local image appears in `crate images`, which blobs it needs, and when it is safe to delete data.
