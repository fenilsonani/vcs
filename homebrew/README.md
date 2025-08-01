# VCS Hyperdrive Homebrew Formula

This directory contains the Homebrew formula for VCS Hyperdrive.

## For Users

### Install VCS Hyperdrive via Homebrew

```bash
# Add the tap
brew tap fenilsonani/vcs

# Install VCS
brew install vcs

# Verify installation
vcs --version
vcs --check-hardware

# Run quick benchmark
vcs benchmark --quick
```

### Update VCS

```bash
brew update
brew upgrade vcs
```

## For Maintainers

### Setting up the Homebrew Tap

1. Create a separate repository for the Homebrew tap:
   ```bash
   # Create new repository: homebrew-vcs
   gh repo create fenilsonani/homebrew-vcs --public
   ```

2. Copy the formula to the tap repository:
   ```bash
   git clone https://github.com/fenilsonani/homebrew-vcs.git
   cd homebrew-vcs
   mkdir -p Formula
   cp ../vcs/homebrew/vcs.rb Formula/
   git add Formula/vcs.rb
   git commit -m "Add vcs formula"
   git push origin main
   ```

3. Test the formula:
   ```bash
   brew tap fenilsonani/vcs
   brew install --build-from-source vcs
   brew test vcs
   ```

### Updating the Formula

The formula is automatically updated by GitHub Actions when a new release is created. Manual updates can be done by:

1. Update the `url` and `sha256` in `Formula/vcs.rb`
2. Test the updated formula
3. Commit and push changes

### Formula Details

- **Name**: `vcs`
- **Description**: VCS Hyperdrive - The World's Fastest Version Control System
- **Homepage**: https://github.com/fenilsonani/vcs
- **License**: MIT
- **Dependencies**: Go (build-time only)

### Supported Platforms

- macOS 10.15+ (Catalina and later)
- Intel x86-64 (with SHA-NI, AVX-512 support)
- Apple Silicon (with NEON optimizations)

### Performance Notes

The Homebrew installation automatically:
- Detects hardware capabilities
- Enables appropriate optimizations
- Provides optimal performance for your Mac
- Includes shell completions

### Troubleshooting

**Installation fails**:
```bash
# Clean and retry
brew uninstall vcs
brew untap fenilsonani/vcs
brew tap fenilsonani/vcs
brew install vcs
```

**Performance issues**:
```bash
# Check hardware detection
vcs --check-hardware

# Run diagnostics
vcs benchmark --quick
```

**Build from source**:
```bash
# Force build from source
brew install --build-from-source vcs
```

### Support

- 📧 Issues: https://github.com/fenilsonani/vcs/issues
- 📖 Documentation: https://github.com/fenilsonani/vcs/tree/main/docs
- 💬 Discussions: https://github.com/fenilsonani/vcs/discussions