# Data Integrity Design for Distributing Files and Blocks

For each distributing file, we record all blocks' checksum into a separate file, which is named by MD5 file. By this MD5 file, Dragonfly guarantees the integrity of distributing files and blocks.

## What is a MD5 file for a block?

Each MD5 file maps each distributing file. The MD5 file includes the following three parts:

- every block's MD5 info;
- the whole distributing file's MD5 info;
- the SHA1 info of the above data;

## How is a MD5 file produced?

Here is the procedure of how to produce a MD5 file in Dragonfly:

1. When dfget agents trigger to distribute a file, Supernode makes the CDN manager to download the target file.
2. As downloading the file, CDN manager will process to generate blocks for target file according to raw of data.
3. Each time finishing to produce a block, the CDN manager would calculate MD5 for the block, and the store the MD5 in memory.
4. When finishing to calculate all MD5 for all blocks, the CDN manager will continue to calculate MD5 for the downloading file.
5. The CDN manager continues to calculate SHA1 of all calculated MD5 values.
6. The CDN managers write **each block's MD5**, **the whole file's MD5** and **SHA1 of all MD5 values** into the block metadata file.

The following is the brief example of a MD5 file:

```
$ ll -h /home/admin/supernode/repo/download/1a0
-rw-r--r-- 1 root root  22M Feb  4 09:23 1a09b5b0c0bb42c2f87b217b71a637c766ee8c598ba3790025d83212f2e7dce1
-rw-r--r-- 1 root root  319 Feb  4 09:23 1a09b5b0c0bb42c2f87b217b71a637c766ee8c598ba3790025d83212f2e7dce1.md5
-rw-r--r-- 1 root root  376 Feb  4 09:23 1a09b5b0c0bb42c2f87b217b71a637c766ee8c598ba3790025d83212f2e7dce1.meta

$ cat 1a09b5b0c0bb42c2f87b217b71a637c766ee8c598ba3790025d83212f2e7dce1.md5
ce5584163a368f2856c0a28cdac1a731:4194304 // MD5 of 1st block and the length
73ac985ba8d37fbc99c3c113c9170e84:4194304 // MD5 of 2nd block and the length
60df935374c85b208c4cb43d1959f331:4194304 // MD5 of 3rd block and the length
35d319f56c4895daa5a9f8427de4340e:4194304 // MD5 of 4th block and the length
b0a07657a0b1b0e133c79785c63e8724:4194304 // MD5 of 5th block and the length
2362dcbf5d5294be0b1744f40f0e423a:1048606 // MD5 of 6th block and the length
4aae22e14d5a70eaa769d3ee50804427         // MD5 of the whole distributing file
418fe37595d7d3f9731a6d6b335b275605bdb791 // SHA1 of all above things
```

## How is a MD5 file consumed?

Here is the procedure of how to consume a MD5 file in Dragonfly:

1. When distributing a block to a peer, supernode will fetch the MD5 of this block and send this MD5 to peer along with block;
2. After finishing to download the block and MD5, dfget will generate another MD5 of the downloaded block, and check the newly-generated MD5 value with downloaded MD5 value.
3. If they match, we can trust the data integrity. If they don't match, dfget will report this mismatch to supernode including block info and incorrect peer, and then dfget will try to download the block from other peers;
4. Supernode gets aware of which peer provided incorrect data integrity, and then supernode isolates this peer from the whole peer network.
