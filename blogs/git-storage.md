see the site here: [git.wyat.me](https://git.wyat.me)

# Pt. 1 - Git's object model is simple

Every developer uses git every day. Very few have looked at how git actually stores on disk.

You can easily see by going into the terminal and typing this:

```bash
echo "hello" | git hash-object --stdin
```

You'll get back `ce013625030ba8dba906f756967f9e9ca394464a`. Every time. On any machine. And there you go.

What is actually being hashed isn't just "hello" but `blob 6\0hello\n`

The type (blob), a space, the content length (6), a null byte (`\0` or `\x00`), then the content (`"hello\n"`). Then just SHA-1 it and you have a git object address. Zlib will then compress it and write it to `.git/objects/ce/013625...`

Here is how it looks in Go:

```go
func Serialize(obj *Object) (compressed []byte, sha string, err error) {
	header := fmt.Sprintf("%s %d\x00", obj.Type, len(obj.Data))
	content := append([]byte(header), obj.Data...)

	sum := sha1.Sum(content)
	sha = hex.EncodeToString(sum[:])

	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err = w.Write(content); err != nil {
		return nil, "", fmt.Errorf("zlib write: %w", err)
	}
	if err = w.Close(); err != nil {
		return nil, "", fmt.Errorf("zlib close: %w", err)
	}

	return buf.Bytes(), sha, nil
}
```

If you call this with `{Type: "blob", Data: []byte("hello\n")}`, you will get back the same SHA that git actually produces.

The parser is also pretty straightforward:

```go
func parse(content []byte) (*Object, error) {
	nullIdx := bytes.IndexByte(content, 0)
	if nullIdx == -1 {
		return nil, fmt.Errorf("invalid object: no null byte")
	}

	header := string(content[:nullIdx])
	data := content[nullIdx+1:]

	parts := strings.SplitN(header, " ", 2)
	// parts[0] = "blob", parts[1] = "6"
	// data = the actual bytes

	size, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid size in header: %w", err)
	}
	if size != len(data) {
		return nil, fmt.Errorf("invalid data size: expected %d, got %d", size, len(data))
	}

	return &Object{Type: ObjectType(parts[0]), Data: data}, nil
}
```

Just find the null byte, split on it, parse header, check size. Git's object parser done in 20 lines.

Git has 4 object types: blobs (file content), trees (directory listings), commits (snapshots with meta data, I call them save points), and tags (named pointers). Every one uses the same format — the complexity lives more in the branching model and reflogs, not the storage.

---

# Pt. 2 - Git is just a content-addressed key-value store

Now that you know the format, the architecture is really pretty simple.

The SHA is the key, zlib-compressed bytes are the value. The store is append-only and immutable — same content always produces the same key, and you never overwrite an existing object. Refs (`HEAD`, `refs/heads/main`, etc) are just separate indexes that map human-readable names to SHA keys.

That is just content-addressed storage. Same idea behind IPFS, Merkle trees in blockchain, and just your normal hash map that you learned in data structures class. The great thing about git is that it applied this simple concept to version control with minimal design.

Every git operation reduces to a small set of key-value primitives:

- `git push` checks which objects the remote already has (`Exists`), then sends the ones that it doesn't (`Put`).
- `git clone` fetches objects by SHA (`Get`).
- `git log` traverses up the chain of commit objects, each pointing to a different SHA.

You can then model your object store after this:

```go
type ObjectStore interface {
	Put(obj *object.Object) (sha string, err error)
	Get(sha string) (*object.Object, error)
	Exists(sha string) (bool, error)
}
```

Just 3 methods — all you need to communicate with a git server. Everything else, like packfiles, deltas, ref negotiation, is just optimization on top of this.

The SHA also doubles as a deduplication mechanism. If two commits include the same file content, they share the same blob in storage. Writes are trivially safe: all 3 backends in this project check `Exists` before writing, and an `INSERT OR IGNORE` in the SQLite implementation handles concurrent writes without conflict.

---

# Pt. 3 - Why does storage choice matter?

So, if it is just a key-value store with 3 simple operations, why not use whatever storage mechanism is easiest? Postgres, S3, NoSQL all support lookup by primary key.

The mismatch is in the access pattern, not the interface.

Git's workload is mainly utilized by small objects at high frequency. A typical `git push` on an active repo triggers hundreds of `Exists` checks per second (one per object being sent). A `git clone` of a large repo can `Get` thousands of small blobs in rapid succession. The objects are typically kilobytes (they are usually just text after all). The bottleneck is per-operation overhead, not bandwidth.

This is where backends diverge quite a bit:

**SQLite** is an embedded database with excellent read performance for sequential workloads. Every `Exists` is a `COUNT(1)` query through the SQL planner — with WAL mode and a serialized write connection to avoid `SQLITE_BUSY` errors under concurrency. I had to set a connection limit of 1, which is not a bug, it was a necessary constraint. It works, but it's not what the engine was designed for.

**BadgerDB** is an LSM-tree key-value store written in Go. There is no impedance mismatch — the data model is SHA→bytes, and BadgerDB's data model is key→value. `Exists` is a single index lookup with no query planner, SQL parsing, or connection pool. Concurrent reads are lock-free by design:

```go
func (s *BadgerStore) Exists(sha string) (bool, error) {
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(sha))
		return err
	})
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	// ...
	return true, nil
}
```

`txn.Get` is the whole operation. No BS.

**MinIO/S3** introduces a network boundary. Every operation crosses the wire, regardless of object size. An `Exists` check on a 40-byte SHA costs a full HTTP `StatObject` round trip to the object storage endpoint.

```go
func (s *MinioStore) Exists(sha string) (bool, error) {
	_, err := s.client.StatObject(
		context.Background(),
		s.bucket,
		sha,
		minio.StatObjectOptions{},
	)
	// ...
}
```

`StatObject` is an HTTP HEAD request. In production on Railway, that's a network hop to a cloud storage API and back. For a 1KB blob or a 1MB blob, the cost is about the same.

---

# Pt. 4 - The results

Each benchmark runs 100 iterations per operation per object size (1KB, 100KB, 1MB), measuring p50 latency, p99 latency, and ops/sec. The results below are from a test on the Railway deployed instance:

**BadgerDB vs S3: 2,668×**

That is the difference between a local index lookup and a blocking network call in a tight loop. During a `git push`, git finds out which objects the server has by calling `Exists` on every candidate object in sequence. At 101 ops/sec, a push with 1,000 new objects takes almost 10 seconds just to check existence. At 270k ops/sec, the same push completes in just under 4ms.

The reason that the gap is so wide is due to the structure of the two data stores. MinIO/S3's `Exists` is implemented as `StatObject` — a HEAD request. Even on a fast network, an HTTP round trip to a managed object storage API has latency that is throttled: DNS, TLS handshake amortization, serialization, server-side processing, response deserialization, etc. In production on Railway, that floor was about 6ms per call. It doesn't matter how fast the storage system is if there are network calls every time.

BadgerDB's `Exists` is an in-process memory-mapped B-tree lookup — that's a mouthful. But that means there is no network. The result is either in cache or a single disk read.

**Concurrent Put at 1KB: 17× difference**

SQLite's single-connection is its real bottleneck here. It serializes all concurrent writes into one stream while BadgerDB's LSM tree handles concurrent writes natively with no constraint. At small object sizes in prod, BadgerDB wins by about 17×.

**Large object convergence**

At 1MB, the gaps start to collapse: SQLite ~36 ops/sec, BadgerDB 48, S3 33. All within noise of each other. At this size, the bottleneck is I/O and compression (packfile optimization), not the storage layer. The backend choice matters most when the objects are small — which is exactly what you already do with git.

**S3 Consistent Latency Floor**

At 1KB, a `Get` on S3 costs ~27ms in production. A 1MB `Get` costs roughly the same (~25ms). The network round trip dominates regardless of payload. Git, as you should know, mainly involves small objects — this floor is always active no matter the size.

---

# Pt. 5 - How GitHub, GitLab, & Bitbucket Solved this at scale

The conclusion from these benchmarks seems obvious: use BadgerDB. At small scale, that can work well enough. But at the scale GitHub & GitLab operate at, the problem is a different beast.

**Native approach fails early**

A single BadgerDB instance is fast but single-tenant. You can't share a BadgerDB store across machines without application-level partitioning. The memory required for a large number of concurrent repos causes cache eviction. This works on a personal git server, but not for millions of repos.

Early hosted git services (including GitHub) ran bare git repos on file systems. Git's native `.git/objects` layout is content-addressed storage with a local file system — which is exactly what I used in this project, just with files instead of an embedded database. It works to a point, but then fails under concurrent access.

**GitLab's answer: Gitaly**

GitLab's response was to build Gitaly, a gRPC service that wraps all git ops behind an RPC interface. Rather than letting application servers make direct filesystem calls to git repos, every git op routes through Gitaly nodes which own the repo data. This decouples the storage layer from the application layer, allows horizontal scaling, and makes it possible to replicate repos across failure zones.

Gitaly effectively takes the `ObjectStore` interface idea from this project and implements it at scale.

**GitHub's evolution**

GitHub's architecture evolved as well, but from a different starting point. Their early system used a distributed file system approach with custom routing that mapped repo names to file server nodes. They eventually built Spokes, a system for replicating git repos to multiple geographically distributed nodes — with writes going to a primary and reads being served closest to the user.

The key insight from GitHub's architecture is that git's content-addressed model makes replication fundamentally easier than it would be for mutable data. Since SHA collisions are extremely unlikely and objects are immutable after they are written, the replicas should never have conflicting versions of the same object.

**The read path matters**

As we've talked about, the read path matters most. `Exists` is called on almost every git operation. That is why the 2,668× difference matters so much. AI will now be creating branches, committing, pulling, etc. at thousands of ops/sec — if it takes 27ms for every one of those operations, it would take weeks for your OpenClaw to make you your next big thing.

---

*You can run the benchmarks yourself at [git.wyat.me](https://git.wyat.me) or clone the source at `git clone https://git.wyat.me/git-storage.git`.*
