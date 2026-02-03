#!/usr/bin/env bash
# Test the Homebrew formula locally.
set -e
brew install --build-from-source Formula/sagasu.rb
brew test sagasu
brew services start sagasu
sleep 2
curl -f http://localhost:8080/health
brew services stop sagasu
brew uninstall sagasu
