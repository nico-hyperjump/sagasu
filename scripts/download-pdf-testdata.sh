#!/usr/bin/env bash
# Download sample PDFs from Mozilla pdf.js test suite (free, open-source test assets).
# Source: https://github.com/mozilla/pdf.js/tree/master/test/pdfs
set -e

DEST="${1:-test/testdata/pdfs}"
BASE_URL="https://raw.githubusercontent.com/mozilla/pdf.js/master/test/pdfs"

# Selection of PDFs: small/simple files that are stored in the repo (not .link externals)
# and are useful for extraction/search tests.
PDFs=(
  "basicapi.pdf"
  "file_pdfjs_test.pdf"
  "doc_1_3_pages.pdf"
  "comments.pdf"
  "find_all.pdf"
  "empty.pdf"
  "Test-plusminus.pdf"
  "S2.pdf"
  "alphatrans.pdf"
  "attachment.pdf"
  "autoprint.pdf"
  "canvas.pdf"
  "dates.pdf"
  "extract_link.pdf"
  "franz.pdf"
  "highlight.pdf"
)

mkdir -p "$DEST"
for name in "${PDFs[@]}"; do
  url="${BASE_URL}/${name}"
  echo "Downloading $name ..."
  curl -sfL -o "${DEST}/${name}" "$url" || echo "  (skip: $name failed)"
done
echo "Done. PDFs in $DEST"
