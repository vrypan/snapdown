# SnapSnapDown (Snapchain Snapshot Downloader)

`snapsnapdown` lets you download a [Snapchain](https://github.com/farcaster_xyz/snapchain) snapshot
before starting a new snapchain node.

![screenshot](screenshot.png)

`snapsnapdown` gives you more flexibility than the embedded downloader:

- You can stop/start and it will pick up where you left.
- When restarting, local chunk sizes will be compared to remote, and if they do not match they will be re-downloaded
- Concurrent chunk downloads: I have found that sometimes a chunk may download at very low speeds, having concurrent downloads removes the bottleneck and results in faster overall download.
- Downloaded chunks are not automatically deleted.

## Usage

1. Download, and unzip [the binary that corresponds to your platform](https://github.com/vrypan/snapsnapdown/releases)
2. Run `./snapsnapdown download` (`./snapsnapdown download --help` for options)


If you want to build from source, clone the repo an type `make`

## After the all chuncks have been downloaded

**The next version of `snapsnapdown` will offer archive extraction too, but until then:**

The downloaded chunks will probably be in `./snapshot/shard-*`, unless you specified a different directory when downloading.

You can test the data integrity using the following command (check with shards 0, 1 2)
```
cat ./snapshot/shard-0/* | tar tzvf -
```

If no error is reported, you can extract the snapshot to `.rocks` like this

```
cat ./snapshot/shard-0/* | tar tzvf - -C .rocks/
```
**Repeat for `shard-1` and `shard-2`!!!**

Now you can start your node and it will pick up syncing where the snapshot left it.

You should probably remove the downloaded chunks with `rm -rf ./snapshot`
