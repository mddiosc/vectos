class Vectos < Formula
  desc "Local-first code context engine for AI agents"
  homepage "https://github.com/mddiosc/vectos"
  version "dev"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/mddiosc/vectos/releases/download/v#{version}/vectos_#{version}_darwin_arm64.tar.gz"
      sha256 "REPLACE_WITH_DARWIN_ARM64_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/mddiosc/vectos/releases/download/v#{version}/vectos_#{version}_linux_amd64.tar.gz"
      sha256 "REPLACE_WITH_LINUX_AMD64_SHA256"
    end
    if Hardware::CPU.arm?
      url "https://github.com/mddiosc/vectos/releases/download/v#{version}/vectos_#{version}_linux_arm64.tar.gz"
      sha256 "REPLACE_WITH_LINUX_ARM64_SHA256"
    end
  end

  def install
    bin.install "vectos"
  end

  test do
    system "#{bin}/vectos"
  end
end
