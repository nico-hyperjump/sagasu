# frozen_string_literal: true

class Sagasu < Formula
  desc "Fast local hybrid search engine combining semantic and keyword search"
  homepage "https://github.com/hyperjump/sagasu"
  url "https://github.com/hyperjump/sagasu/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256"
  license "MIT"
  head "https://github.com/hyperjump/sagasu.git", branch: "main"

  depends_on "go" => :build
  depends_on "onnxruntime"
  depends_on "pkg-config" => :build

  def install
    ENV["CGO_ENABLED"] = "1"
    system "go", "build",
           *std_go_args(ldflags: "-s -w -X main.version=#{version}"),
           "./cmd/sagasu"

    (etc/"sagasu").install "config.yaml.example" => "config.yaml"
    (var/"sagasu/data/models").mkpath
    (var/"sagasu/data/indices/bleve").mkpath
    (var/"sagasu/data/indices/faiss").mkpath
    (var/"sagasu/data/db").mkpath
  end

  def post_install
    model_path = var/"sagasu/data/models/all-MiniLM-L6-v2.onnx"
    unless model_path.exist?
      ohai "Downloading embedding model (one-time, ~80MB)..."
      system "curl", "-L",
        "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx",
        "-o", model_path
      ohai "Model downloaded successfully!"
    end

    config_file = etc/"sagasu/config.yaml"
    if config_file.exist?
      inreplace config_file do |s|
        s.gsub! "/usr/local/var/sagasu", var/"sagasu"
        s.gsub! "/usr/local/etc/sagasu", etc/"sagasu"
      end
    end
  end

  service do
    run [opt_bin/"sagasu", "server", "--config", etc/"sagasu/config.yaml"]
    keep_alive true
    log_path var/"log/sagasu.log"
    error_log_path var/"log/sagasu.log"
    working_dir var/"sagasu"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/sagasu version")
    system bin/"sagasu", "help"
  end
end
