# Chapter 1 - Image References and Pulling

Crate is written in Go, but the first problem it solves is not Go-specific. Before a runtime can isolate a process, it needs bytes on disk. Pulling is the step that turns a human-friendly name like `alpine` into concrete image blobs.

## The Problem

An image reference is not a filesystem. It is just a name that has to be resolved through a registry protocol into immutable content.

The gap looks like this:

- humans type `alpine`
- registries store manifests and blobs keyed by digest
- the runtime needs local files, not symbolic names

The important design idea is that tags are mutable and digests are not. If a runtime treated tags as content identities, reproducibility would collapse immediately.

> Under the Hood
>
> Container image pulling is a naming problem first and an I/O problem second. Most of the complexity comes from translating a user-facing name into a content graph safely.

## How Linux Does It

Linux itself does not know about container registries. Pulling is ordinary userspace networking and file I/O.

You can see the same flow with raw HTTP:

```sh
repo=library/alpine
tag=latest

token=$(curl -fsSL \
  "https://auth.docker.io/token?service=registry.docker.io&scope=repository:${repo}:pull" \
  | jq -r .token)

curl -fsSL \
  -H "Authorization: Bearer $token" \
  -H "Accept: application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.v2+json" \
  "https://registry-1.docker.io/v2/${repo}/manifests/${tag}"
```

That shell snippet is not "container magic". It is just HTTPS plus JSON.

## How Crate Uses It

Crate normalizes references first, resolves the current tag at the registry, then checks whether the resolved manifest is already present locally.

```go
// internal/image/reference.go
ref := &Reference{
    Registry: "docker.io",
    Tag:      "latest",
}

if !strings.Contains(repo, "/") {
    repo = "library/" + repo // "alpine" becomes "library/alpine"
}
```

The pull path itself is intentionally linear:

```go
// internal/image/pull.go
imgRef, err := ParseReference(input)
if err != nil {
    return err
}

manifestData, contentType, manifestDigest, err := fetchManifestByTag(imgRef)
if err != nil {
    return err
}

img, err := resolveManifest(imgRef, manifestData, contentType, manifestDigest)
if err != nil {
    return err
}

if current, err := ReadMetadata(imgRef); err == nil && current.ManifestDigest == img.ManifestDigest {
    fmt.Println("Image already present")
    return nil
}
```

After resolution, Crate stores one JSON file per manifest digest under `~/.local/share/crate/images/`. Tags live inside that file as `repoTags`, so a local lookup scans manifest records and finds the one that owns `alpine:latest` or whatever tag the user asked for.

If a tag moves to a different manifest later, Crate removes that tag from the old manifest record and adds it to the new one. If the old manifest becomes untagged, Crate deletes its metadata file and prunes its config and layer blobs only when no other local manifest still references them.

The equivalent raw Linux approach is just "fetch JSON, then download files", but Crate adds a small amount of policy and local bookkeeping:

- short-name normalization in [`internal/image/reference.go`](../internal/image/reference.go)
- resolved-manifest comparison in [`internal/image/pull.go`](../internal/image/pull.go)
- manifest-backed metadata in [`internal/image/metadata.go`](../internal/image/metadata.go)
- digest-addressed blob storage in the image package

> ⚠ Watch out
>
> `latest` is a convenience tag, not a stable identity. Crate re-resolves the tag on every pull and only skips work when the resolved manifest digest matches local metadata.

## Connecting the Dots

This chapter gets bytes into the local image store. The next chapter explains how Crate decides which exact bytes belong to a tag in the first place, which is the job of manifests and platform selection.

## Try It Yourself

Run:

```sh
go run ./cmd/crate pull alpine
go run ./cmd/crate pull alpine
```

Then inspect the local image store and compare the first run to the second. The first run should resolve and download; the second should still contact the registry to resolve `latest`, but it should stop once the local manifest digest matches.

## Key Takeaways

- Image pulling starts from a mutable name and ends with immutable local content.
- Linux provides the networking and file primitives; the registry protocol lives in userspace.
- Crate stores local image metadata by manifest digest, not by tag filename.
- Crate keeps reference parsing and pull policy explicit so the flow is easy to follow.
- Pulling is only the front door; manifests decide what content actually belongs to a tag.
