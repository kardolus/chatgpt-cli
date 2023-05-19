class ChatgptCli < Formula
  desc "A CLI for the OpenAI ChatGPT API"
  homepage "https://github.com/kardolus/chatgpt-cli"
  
  if Hardware::CPU.intel?
    url "https://github.com/kardolus/chatgpt-cli/releases/download/v1.0.4/chatgpt-darwin-amd64"
    sha256 "bc0db3d33c1cefb1ee914435129a5dd85138a03945790d2e826e52923d483b4b"
  elsif Hardware::CPU.arm?
    url "https://github.com/kardolus/chatgpt-cli/releases/download/v1.0.4/chatgpt-darwin-arm64"
    sha256 "c3df40f107bc1df45f72780957fef47c12df167dc317898d85ab0e95a47ab542"
  end

  def install
    bin.install "chatgpt"
  end

  test do
    system "#{bin}/chatgpt", "--help"
  end
end

