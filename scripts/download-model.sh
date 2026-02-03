#!/usr/bin/env bash
# Download the all-MiniLM-L6-v2 ONNX model for Sagasu.
set -e
MODEL_DIR="${1:-/usr/local/var/sagasu/data/models}"
mkdir -p "$MODEL_DIR"
MODEL_PATH="$MODEL_DIR/all-MiniLM-L6-v2.onnx"
if [ -f "$MODEL_PATH" ]; then
  echo "Model already exists: $MODEL_PATH"
  exit 0
fi
echo "Downloading embedding model to $MODEL_PATH..."
curl -L "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/onnx/model.onnx" -o "$MODEL_PATH"
echo "Download complete."
