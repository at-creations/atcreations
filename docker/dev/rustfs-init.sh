#!/bin/sh
# Creates the dev media bucket in RustFS and makes image objects publicly
# readable, mirroring a CDN in front of R2: anonymous GET of objects under
# products/ and categories/ only — no anonymous listing, upload or delete.
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
    \"Resource\": [\"arn:aws:s3:::${BUCKET}/products/*\", \"arn:aws:s3:::${BUCKET}/categories/*\"]
  }]
}"
echo "public-read policy applied to products/ and categories/"
