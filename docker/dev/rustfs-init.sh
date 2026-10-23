#!/bin/sh
# Creates the dev media bucket in RustFS and makes image objects publicly
# readable, mirroring a CDN in front of R2: anonymous GET of objects under
# products/, categories/ and site/ only — no anonymous listing, upload or
# delete. Marketing images in /site-assets (docker/dev/site-assets, filled by
# fetch-site-assets.sh) are synced to site/.
set -eu
S3="aws --endpoint-url ${S3_ENDPOINT} s3api"

if $S3 head-bucket --bucket "$BUCKET" >/dev/null 2>&1; then
  echo "bucket $BUCKET exists"
else
  $S3 create-bucket --bucket "$BUCKET" >/dev/null
  echo "created bucket $BUCKET"
fi

$S3 put-bucket-policy --bucket "$BUCKET" --policy "{
  \"Version\": \"2012-10-17\",
  \"Statement\": [{
    \"Sid\": \"PublicReadImages\",
    \"Effect\": \"Allow\",
    \"Principal\": \"*\",
    \"Action\": [\"s3:GetObject\"],
    \"Resource\": [
      \"arn:aws:s3:::${BUCKET}/products/*\",
      \"arn:aws:s3:::${BUCKET}/categories/*\",
      \"arn:aws:s3:::${BUCKET}/site/*\"
    ]
  }]
}"
echo "public-read policy applied to products/, categories/ and site/"

if [ -d /site-assets ] && ls /site-assets/*.jpg /site-assets/*.png >/dev/null 2>&1; then
  aws --endpoint-url "${S3_ENDPOINT}" s3 sync /site-assets "s3://${BUCKET}/site/" \
    --exclude "*" --include "*.jpg" --include "*.png" \
    --cache-control "public, max-age=3600" --only-show-errors
  echo "synced marketing images to site/"
else
  echo "no marketing images in docker/dev/site-assets (run docker/dev/fetch-site-assets.sh)"
fi
