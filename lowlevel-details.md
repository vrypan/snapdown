# Low Level Details about Snapshots

I had to reverse engineer how snapshots are downloaded by nodes on startup,
so here's how it works in case you want to implement something on your
own.

`endpointURL = "https://pub-d352dd8819104a778e20d08888c5a661.r2.dev"`

This URL is hard-coded [in the node code](https://github.com/farcasterxyz/snapchain/blob/f82a7e559711deac60b819c4d92bad1aaca55946/src/storage/db/snapshot.rs#L55).

## Setp 1
For each shard (0,1,2), you fetch

```
<endpointURL>/FARCASTER_NETWORK_MAINNET/<shard id>/latest.json
```

It looks like this (shards 1 and 2 have ~1600 chunks):

```json
{
  "key_base": "FARCASTER_NETWORK_MAINNET/0/snapshot-2025-06-24-1750741283.tar.gz",
  "chunks": [
    "chunk_0001.bin",
    "chunk_0002.bin",
    "chunk_0003.bin",
    "chunk_0004.bin",
    "chunk_0005.bin",
    "chunk_0006.bin",
    "chunk_0007.bin",
    "chunk_0008.bin",
    "chunk_0009.bin",
    "chunk_0010.bin",
    "chunk_0011.bin",
    "chunk_0012.bin",
    "chunk_0013.bin",
    "chunk_0014.bin",
    "chunk_0015.bin",
    "chunk_0016.bin",
    "chunk_0017.bin",
    "chunk_0018.bin",
    "chunk_0019.bin",
    "chunk_0020.bin",
    "chunk_0021.bin",
    "chunk_0022.bin",
    "chunk_0023.bin",
    "chunk_0024.bin",
    "chunk_0025.bin",
    "chunk_0026.bin",
    "chunk_0027.bin",
    "chunk_0028.bin",
    "chunk_0029.bin",
    "chunk_0030.bin",
    "chunk_0031.bin",
    "chunk_0032.bin",
    "chunk_0033.bin",
    "chunk_0034.bin",
    "chunk_0035.bin",
    "chunk_0036.bin",
    "chunk_0037.bin",
    "chunk_0038.bin",
    "chunk_0039.bin",
    "chunk_0040.bin",
    "chunk_0041.bin",
    "chunk_0042.bin"
  ],
  "timestamp": 1750741283744
}
```

## Step 2

You can download each chunk from
```
<endpointURL>/<key_base>/<chunk_name>
```

For example:

```
https://pub-d352dd8819104a778e20d08888c5a661.r2.dev/FARCASTER_NETWORK_MAINNET/0/snapshot-2025-06-24-1750741283.tar.gz/chunk_0001.bin
```

If you download all the chunks of a shard, you can combine them into the originating tar.gz archive:

```
cat chunk_* >> snapshot-2025-06-24-1750741283.tar.gz
```

The contents of this archive look like this:

```
cat snapshot/shard-0/* | tar tzvf -
drwxr-xr-x  0 0      0           0 Jun 20 08:01 shard-0/
-rw-r--r--  0 0      0     3840673 Jun 20 08:01 shard-0/000101.sst
-rw-r--r--  0 0      0    10495619 Jun 20 08:01 shard-0/000100.sst
-rw-r--r--  0 0      0    10495657 Jun 20 08:01 shard-0/000099.sst
-rw-r--r--  0 0      0    10499238 Jun 20 08:01 shard-0/000098.sst
-rw-r--r--  0 0      0    10503563 Jun 20 08:01 shard-0/000097.sst
-rw-r--r--  0 0      0    10493235 Jun 20 08:01 shard-0/000096.sst
-rw-r--r--  0 0      0    18293840 Jun 20 08:01 shard-0/000095.sst
...
-rw-r--r--  0 0      0    51284244 Jun 20 08:00 shard-0/000012.sst
-rw-r--r--  0 0      0    49125619 Jun 20 08:00 shard-0/000011.sst
-rw-r--r--  0 0      0    47281951 Jun 20 08:00 shard-0/000010.sst
-rw-r--r--  0 0      0    46139117 Jun 20 08:00 shard-0/000009.sst
-rw-r--r--  0 0      0    51600025 Jun 20 08:00 shard-0/000008.sst
-rw-r--r--  0 0      0        7248 Jun 20 08:00 shard-0/OPTIONS-000007
-rw-r--r--  0 0      0       33753 Jun 20 08:01 shard-0/MANIFEST-000005
-rw-r--r--  0 0      0           0 Jun 20 08:00 shard-0/000004.log
-rw-r--r--  0 0      0          16 Jun 20 08:00 shard-0/CURRENT
-rw-r--r--  0 0      0          36 Jun 20 08:00 shard-0/IDENTITY
-rw-r--r--  0 0      0           0 Jun 20 08:00 shard-0/LOCK
-rw-r--r--  0 0      0      422110 Jun 20 08:01 shard-0/LOG
```

The above is in practice `.rocks/shard-0` on a node. So, extracting the snapshot archives
in `.rocks` will give you a node at the same state as the node that generated the snapshot.
