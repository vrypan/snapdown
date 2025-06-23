# SnapSnapDown (Snapchain Snapshot Downloader)

`snapsnapdown` lets you download a [Snapchain](https://github.com/farcaster_xyz/snapchain) snapshot
before starting a new snapchain node.

![screenshot](screenshot.png)

It is intended to replace the embedded downloader:

- You can stop/start and it will pick up where you left.
- When restarting, local chunk sizes will be compared to remote, and if they do not match they will be re-downloaded
- Concurrent chunk downloads: I have found that sometimes a chunk may download at very low speeds, having concurrent downloads removes the bottleneck and results in faster overall download.

## Usage

1. Download, and unzip the [binary that corresponds to your platform](https://github.com/vrypan/snapsnapdown/releases)
2. run `./snapsnapdown --help` or `./snapsnapdown download --help`

If you want to build from source, clone the repo an type `make`
