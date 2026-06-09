# Images And Storage

Before a container can start, Crate needs files. An image name like `alpine` is only a convenient handle. The runtime must resolve that name through a registry, download immutable blobs, record local metadata, and later unpack those blobs into a container root filesystem.

This part covers the image side of that work.

If we have only used Docker from the command line, it can be tempting to think of an image as one big archive. It is more useful to think of it as a small graph of content:

```text
tag -> manifest -> config
                -> layer
                -> layer
                -> layer
```

The tag is the friendly name. The manifest is the document that says which content belongs to the image. The config describes how the image wants to run. The layers contain filesystem changes.

Crate's image package exists to turn that graph into local files that the rest of the runtime can use.

## Beginner Terms

An image is a template for creating a container. It is not a running thing by itself.

A registry is a server that stores images. Docker Hub is the registry Crate currently knows how to pull from.

A tag is a mutable name such as `latest` or `24.04`. It is convenient for humans but not a stable content identity.

A digest is a content-based identifier such as `sha256:...`. If the content changes, the digest changes.

A manifest is a JSON document that points at the config and layer blobs.

A blob is a downloaded chunk of content stored by digest. In Crate, config blobs and layer blobs both live in the local blob store.

A layer is a compressed tar archive containing filesystem changes. Applying all layers in order gives a container root filesystem.

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
