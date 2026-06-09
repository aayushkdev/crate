# Images And Storage

Before a container can start, Crate needs files. An image name like `alpine` is only a convenient handle. The runtime must resolve that name through a registry, download immutable blobs, record local metadata, and later unpack those blobs into a container root filesystem.

This part covers the image side of that work.

The main implementation files are:

* `internal/image/reference.go`
* `internal/image/registry.go`
* `internal/image/manifest.go`
* `internal/image/pull.go`
* `internal/image/metadata.go`
* `internal/image/store.go`
* `internal/image/remove.go`
* `internal/image/prune.go`
* `internal/fs/layer.go`
* `internal/storage/paths.go`
* `internal/storage/store.go`

## What Happens During pull

At a high level, `crate pull alpine` does this:

1. Parse `alpine` into a full image reference.
2. Ask Docker Hub for a pull token.
3. Fetch the manifest for the tag.
4. If the response is an index, choose the Linux amd64 manifest.
5. Download the config blob and layer blobs.
6. Write local image metadata keyed by manifest digest.
7. Merge repo tags if the same manifest already exists locally.

The important distinction is between names and content. Tags are names and can move. Digests identify content and should not move.

## Chapters

* [Image References And Pulling](01_Image_References_And_Pulling.md)
* [Manifests, Blobs And Metadata](02_Manifests_Blobs_And_Metadata.md)
* [Layer Extraction And Image Pruning](03_Layer_Extraction_And_Image_Pruning.md)

