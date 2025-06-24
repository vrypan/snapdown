# SnapSnapDown (Snapchain Snapshot Downloader)

`snapsnapdown` lets you download a [Snapchain](https://github.com/farcaster_xyz/snapchain) snapshot
before starting a new snapchain node.

![screenshot](screenshot.png)

`snapsnapdown` gives you more flexibility than the embedded downloader:

- You can stop/start and it will pick up where you left.
- When restarting, local chunk sizes will be compared to remote, and if they do not match they will be re-downloaded
- Concurrent chunk downloads: I have found that sometimes a chunk may download at very low speeds, having concurrent downloads removes the bottleneck and results in faster overall download.
- Downloaded chunks are not automatically deleted.

## 1. Install

Download, and unzip [the binary that corresponds to your platform](https://github.com/vrypan/snapsnapdown/releases)

Move it to a directoryt in your `$PATH`. 

If you want to build from source, clone the repo an run `make`

Check: `snapsnapdown version`

## 2. Download the snapshot

Use `snapsnapdown download` to download the snapshot arhive to `./snapshot`. Or `snapsnapdown download --help` for more options.


## 3. Extract the snapshot

The snapshot must be extracted to the `.rocks` directory relative to where you will run your docker container.

Use something like (adjust paths, if your setup is different)

```
snapsnapdown extract ./snapshot .rocks
```


### Extract manually

You can manually extract the archive, using `tar` if you prefer. 

You can test the data integrity using the following command (check with shards 0, 1 2)
```
cat ./snapshot/shard-0/* | tar tzvf -
```

If no error is reported, you can extract the snapshot to `.rocks` like this (and repeat for shards 1 and 2!!!)

```
cat ./snapshot/shard-0/* | tar tzvf - -C .rocks/
```

## 4. After extractiing the snapshot

Now you can start your node and it will pick up syncing where the snapshot left it.

You will probably want to remove the downloaded chunks with `rm -rf ./snapshot` to free space on your disk.
