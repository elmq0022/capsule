# Notes

## Create a BusyBox rootfs

Use Docker to create a container from the `busybox` image, export its filesystem, and unpack it into Capsule's rootfs directory:

```bash
mkdir -p ~/.local/share/capsule/rootfs/busybox
cid=$(docker create busybox)
docker export "$cid" | tar -x -C ~/.local/share/capsule/rootfs/busybox
docker rm "$cid"
```

This works because `docker export` produces the container's merged root filesystem as a tar stream. Extracting that tarball gives Capsule a directory tree that can be used later as the target for `chroot`.

Check the extracted rootfs with:

```bash
ls ~/.local/share/capsule/rootfs/busybox
```

For a quick manual validation outside Capsule, you can try:

```bash
sudo chroot ~/.local/share/capsule/rootfs/busybox /bin/sh
```

BusyBox is intentionally minimal, so expect a small userspace and only a limited command set.
