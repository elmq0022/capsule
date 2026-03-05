# Capsule Design Document

## Overview

## Goals

- The reads will concurrent and the writes will be serialized
  - This will be enforced with a `sync.RWMutex` around the in-memory index
- The storage API is minimal of Get, Set, and Del
- The implementation is a go library and no server or encode/decode protocol
- The storage format will be an append-only segmented log
- The compaction will minimally block normal reads and writes
  - Compaction can be triggered manually or based on file size configuration
- The index will be stored in memory only
- Both keys and values are both stored as bytes
- Durability will be enforced with ack after fsync and fsync after every write
- Crash Consistency
  - Only the acknowledged writes will be guaranteed as durable
  - The record data integrity will be verified by a leading checksum over the header and payload
  - strict CRC + truncate-at-first-bad

## Architecture Invariants

<!-- todo - this section needs work -->

Serial writes to the storage log.
Index contains file number, offset and value size.
Index can always be rebuilt from files.
The storage API is unavailable during index rebuild on startup and crash recovery

## Components

## Storage Format

| CRC | key size | value size | key | value |

## Future Enhancements
