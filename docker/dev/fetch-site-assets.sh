#!/usr/bin/env bash
# Downloads the storefront's marketing images (hero, about, workshop photos,
# logos) from the current production CDN into docker/dev/site-assets/, using
# the new file names. rustfs-init uploads them to the dev bucket under site/.
#
#   bash docker/dev/fetch-site-assets.sh            # skip files already present
#   FORCE=1 bash docker/dev/fetch-site-assets.sh    # re-download everything
#
# The folder is git-ignored; these are business assets, not source code.
set -euo pipefail

SOURCE="${SOURCE:-https://cdn.atcreations.ca/data}"
DEST="$(cd "$(dirname "$0")" && pwd)/site-assets"
mkdir -p "$DEST"

# new-name  old-name (on the old CDN)
MAP="
hero.jpg             hero.jpg
about_row.jpg        about_row.jpg
about_col.jpg        about_col.jpg
pottery.jpg          pottery.jpg
wood-craft.jpg       wood-craft.jpg
workshop-space.jpg   workshop-space.jpg
plaster-models.jpg   plaster-models.jpg
finished-works.jpg   finished-works.jpg
painting-session.jpg painting-session.jpg
logo-nav.png         blue_nav_logo.png
logo-hero.png        white_long_shadow.png
logo-square.png      blue_square.png
og-image.png         og_image.png
"

# Accept only real JPEG/PNG files (checked by magic bytes, not by name).
is_image() {
  local sig
  sig="$(head -c 8 "$1" | od -An -tx1 | tr -d ' \n')"
  case "$sig" in
    ffd8ff*) return 0 ;;          # JPEG
    89504e470d0a1a0a) return 0 ;; # PNG
    *) return 1 ;;
  esac
}

echo "$MAP" | while read -r new old; do
  [ -n "$new" ] || continue
  out="$DEST/$new"
  if [ -s "$out" ] && [ -z "${FORCE:-}" ]; then
    echo "skip  $new (already present)"
    continue
  fi
  tmp="$out.part"
  curl -fsSL --max-time 120 --max-filesize 20000000 -o "$tmp" "$SOURCE/$old"
  if ! is_image "$tmp"; then
    rm -f "$tmp"
    echo "error $old is not a JPEG/PNG; skipped" >&2
    continue
  fi
  mv "$tmp" "$out"
  echo "ok    $new  <- $old ($(wc -c < "$out" | tr -d ' ') bytes)"
done

echo "Done. Restart the stack (or: docker compose -f docker-compose.dev.yml up rustfs-init) to upload them."
