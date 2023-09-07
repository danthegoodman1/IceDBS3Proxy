# example_control_plane

To install dependencies:

```bash
bun install
```

To run:

```bash
bun run index.ts
```

```
curl -X POST http://localhost:8888/resolve_virtual_bucket -d '{"VirtualBucket": "hey"}' -H 'content-type: application/json'
```