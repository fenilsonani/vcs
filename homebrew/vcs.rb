class Vcs < Formula
  desc "VCS Hyperdrive - The World's Fastest Version Control System"
  homepage "https://github.com/fenilsonani/vcs"
  url "https://github.com/fenilsonani/vcs/archive/refs/heads/main.tar.gz"
  version "1.0.0-dev"
  sha256 :no_check # Development version, will be updated for releases
  license "MIT"
  head "https://github.com/fenilsonani/vcs.git", branch: "main"

  depends_on "go" => :build

  def install
    # Set build variables
    ldflags = %W[
      -s -w
      -X github.com/fenilsonani/vcs/cmd/vcs.Version=#{version}
      -X github.com/fenilsonani/vcs/cmd/vcs.BuildTime=#{time.iso8601}
    ]

    # Build the binary
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/vcs"

    # Install shell completions
    generate_completions_from_executable(bin/"vcs", "completion")

    # Install man page
    man1.install "docs/vcs.1" if File.exist?("docs/vcs.1")
  end

  def caveats
    <<~EOS
      VCS Hyperdrive has been installed! 🚀

      To get started:
        vcs --version
        vcs --check-hardware

      For maximum performance on Apple Silicon:
        - NEON optimizations are enabled by default
        - Hardware acceleration will be auto-detected
        - Use 'vcs benchmark --quick' to test performance

      Documentation: https://github.com/fenilsonani/vcs/tree/main/docs
      
      Enjoy 1000x+ faster version control! ⚡
    EOS
  end

  test do
    # Test basic functionality
    system "#{bin}/vcs", "--version"
    system "#{bin}/vcs", "--help"
    
    # Test hardware detection
    system "#{bin}/vcs", "--check-hardware"
    
    # Test repository initialization
    testpath_repo = testpath/"test_repo"
    testpath_repo.mkpath
    Dir.chdir(testpath_repo) do
      system "#{bin}/vcs", "init"
      assert_predicate testpath_repo/".vcs", :exist?
    end
  end
end