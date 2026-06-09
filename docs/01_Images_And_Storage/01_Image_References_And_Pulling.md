# Image References And Pulling

An image reference is the first place where user convenience has to become runtime precision.

A user may type:

```text
alpine
ubuntu:24.04
library/nginx
docker.io/library/busybox:latest
```

Crate normalises this into a registry, repository and tag. If no registry is provided, it assumes `docker.io`. If the repository has no slash, it adds `library/`. If no tag is provided, it uses `latest`.

So:

```text
alpine
```

becomes:

```text
docker.io/library/alpine:latest
```

This logic lives in `internal/image/reference.go`.

## Why Tags Are Not Enough

Tags are mutable. The registry owner can move `latest` from one manifest to another. If Crate treated a tag as the identity of image content, repeated runs could silently use different files while appearing to use the same image.

Crate therefore resolves the tag to a manifest digest during pull. The digest is the stable local identity. The repo tag is stored as a name pointing at that metadata.

This is why the second `crate pull alpine` still contacts the registry. Crate has to ask what `latest` points at today before it can know whether the local manifest is still current.

## Docker Hub Flow

The registry code currently supports Docker Hub-style pulls. The flow in `internal/image/registry.go` is ordinary HTTPS:

1. Request a bearer token from `https://auth.docker.io/token`.
2. Ask for the manifest from `https://registry-1.docker.io/v2/<repo>/manifests/<tag>`.
3. Send an `Accept` header covering OCI manifests, Docker manifests, OCI indexes and Docker manifest lists.
4. Read the response body, content type and `Docker-Content-Digest` header.

There is no Linux container magic here yet. This is userspace networking and JSON parsing. The container-specific part starts later, when layers are applied into a root filesystem and the child process is moved into it.

## Failure Points

The pull path can fail when:

* the registry is not `docker.io`;
* Docker Hub auth fails;
* the tag does not exist;
* the registry returns an unsupported manifest type;
* the image index does not contain a Linux amd64 image;
* a blob download fails;
* local metadata or blob paths cannot be written.

Keeping these errors near the image package is important. Container creation should not need to know whether an image came from a tag, digest, index or single manifest. It should only need local metadata and blobs.

## A More Detailed Walkthrough

Let's follow the simplest input:

```sh
crate pull alpine
```

The user probably thinks of this as "download Alpine". Crate has to treat it as a more precise task:

```text
download the content currently named by docker.io/library/alpine:latest
```

That wording is intentionally careful. The tag is not the content. It is only the current registry pointer to a manifest.

The pull function in `internal/image/pull.go` starts by parsing the reference. It then prints a small progress message and fetches the manifest by tag. At this point Crate still does not know whether the response is the actual image manifest or an index of platform-specific manifests.

The image package resolves that document into one internal shape:

```text
manifest digest
config digest
layer digests
platform
```

Only after this resolution does Crate compare local metadata. This ordering matters. Comparing the user tag before resolving it would only tell us that the same name was requested. It would not tell us that the same content is still behind the name.

## Why Pull Always Resolves First

A common mistake when implementing a small image store is to check whether `alpine:latest` exists locally and skip the registry request if it does. That is fast, but it makes the local tag stale forever.

Crate instead asks the registry first:

```text
what manifest digest does this tag point at now?
```

Then it asks the local store:

```text
do we already have metadata for this tag with that manifest digest?
```

If the answer is yes, the pull can stop. If the digest changed, Crate downloads the new blobs, writes new metadata, and removes the moved tag from the old manifest metadata.

This behavior keeps tags convenient without pretending that they are immutable.

## What Crate Does Not Parse Yet

The reference parser is intentionally small. It handles the forms Crate currently needs, but it is not a complete Docker reference implementation.

Some things a more complete implementation would need to handle include:

* explicit digest references like `alpine@sha256:...`;
* registry ports in all edge cases;
* non-Docker-Hub registry authentication;
* official reference grammar validation;
* better error messages for malformed names.

For a learning runtime, it is reasonable to start with the common path. The important part is that the parser's output is explicit enough for the registry and metadata code.

## Registry Requests Are Still Userspace Work

It is worth pausing on this point: pulling an image does not require a special kernel feature.

The registry side of containers is mostly:

* HTTP requests;
* bearer tokens;
* JSON manifests;
* digest-addressed files;
* local metadata.

Linux container-specific work starts later. Namespaces, mounts and `execve` do not appear during registry resolution. This separation is useful when reading Crate because it prevents us from mixing two different domains:

* distribution, which gets bytes onto disk;
* runtime isolation, which starts a process with a different view of the system.

Crate keeps those domains in different packages.
