class GitAttest < Formula
  desc "Transparency certificates for AI-assisted code contributions"
  homepage "https://github.com/SaaSAILabs/attest-cli"
  url "https://github.com/SaaSAILabs/attest-cli/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin/"git-attest", "main.go"
  end

  # This is the magic: after the binary is installed on PATH,
  # automatically run `git-attest init` to set up global hooks.
  # The developer never has to run a second command.
  def post_install
    system bin/"git-attest", "init"
  end

  def caveats
    <<~EOS
      git-attest is now active on every repository on this machine.

      Your workflow:
        git commit -m "message"              → evidence captured automatically
        git push                             → flight recordings pushed automatically
        git attest push origin feature-x     → for explicit branch pushes

      To disable temporarily:  ATTEST_DEV_MODE=1 git commit ...
      To uninstall completely: git attest uninstall
    EOS
  end

  test do
    system bin/"git-attest", "--help"
  end
end
