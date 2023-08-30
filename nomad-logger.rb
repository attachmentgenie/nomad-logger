# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class NomadLogger < Formula
  desc ""
  homepage ""
  version "0.1.0"
  depends_on :macos

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/attachmentgenie/nomad-logger/releases/download/v0.1.0/nomad-logger_0.1.0_darwin_arm64.zip"
      sha256 "47cff2b8b66c366e0014d21af4713fd5df4db152eb27b60522d2b6fba8e21fd2"

      def install
        bin.install "nomad-logger"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/attachmentgenie/nomad-logger/releases/download/v0.1.0/nomad-logger_0.1.0_darwin_amd64.zip"
      sha256 "55b0ec8648b9f8bb2b2eda509d4beddd3b0668bac19d2b315bfda03db040d0fe"

      def install
        bin.install "nomad-logger"
      end
    end
  end
end