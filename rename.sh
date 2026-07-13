#!/usr/bin/env bash
# Ganti module path boilerplate ke module path baru saat clone jadi service baru.
# Pakai: ./rename.sh github.com/julles/order-service
set -euo pipefail

if [ $# -ne 1 ]; then
	echo "Pakai: ./rename.sh <module-path-baru>"
	echo "Contoh: ./rename.sh github.com/julles/order-service"
	exit 1
fi
NEW="$1"

# Module path lama dibaca dari go.mod (bukan hardcode) agar tetap benar setelah rename.
OLD=$(go mod edit -json | sed -n 's/.*"Path": "\([^"]*\)".*/\1/p' | head -1)
if [ -z "$OLD" ]; then
	echo "Gagal membaca module path dari go.mod"
	exit 1
fi

if [ "$NEW" = "$OLD" ]; then
	echo "Module path baru sama dengan yang lama, tidak ada yang diubah."
	exit 0
fi

# Ganti di go.mod dan semua file .go.
grep -rl --include='*.go' --include='go.mod' "$OLD" . | while read -r f; do
	sed -i "s|$OLD|$NEW|g" "$f"
done

go mod tidy

echo "Selesai: $OLD -> $NEW"
