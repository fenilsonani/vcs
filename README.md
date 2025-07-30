# VCS - High-Performance Git Implementation

VCS is a custom implementation of Git built in Go, focusing on performance optimization and seamless GitHub integration.

## Features

- **Core Git Objects**: Support for blobs, trees, commits, and tags
- **SHA-1 Hashing**: Content-addressable storage system
- **Zlib Compression**: Efficient storage of objects
- **Performance Optimized**: Built with performance in mind
- **GitHub Integration**: (Coming soon) OAuth 2.0 and API integration

## Installation

```bash
# Clone the repository
git clone https://github.com/fenilsonani/vcs.git
cd vcs

# Build the project
make build

# Or install directly
make install
```

## Usage

### Initialize a repository
```bash
vcs init
# or
vcs init /path/to/repo
```

### Hash an object
```bash
# Just compute hash
echo "Hello, World!" | vcs hash-object --stdin

# Write object to repository
echo "Hello, World!" | vcs hash-object -w --stdin

# Hash a file
vcs hash-object -w README.md
```

### Read objects
```bash
# Show object content
vcs cat-file -p <object-id>

# Show object type
vcs cat-file -t <object-id>

# Show object size
vcs cat-file -s <object-id>
```

## Development

### Running tests
```bash
make test

# With coverage
make test-coverage
```

### Running benchmarks
```bash
make bench
```

### Code formatting and linting
```bash
make fmt
make lint
```

## Architecture

The project is structured as follows:

- `cmd/vcs/` - CLI commands
- `internal/core/objects/` - Core git object implementation
- `pkg/vcs/` - Public API for repository operations
- `test/` - Integration tests
- `benchmarks/` - Performance benchmarks

## Performance Features

- In-memory object caching
- Efficient zlib compression
- Optimized SHA-1 hashing
- Parallel processing support (coming soon)
- Memory-mapped files for large objects (coming soon)

## Roadmap

- [x] Core object model (blob, tree, commit, tag)
- [x] SHA-1 hashing and object identification
- [x] Loose object storage with compression
- [x] Basic CLI commands (init, hash-object, cat-file)
- [ ] Index management
- [ ] Working directory operations
- [ ] References (branches, tags)
- [ ] Packfile implementation
- [ ] Remote operations
- [ ] GitHub API integration
- [ ] Performance optimizations

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.