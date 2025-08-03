#!/bin/bash

REMOTE_USER="root"
REMOTE_HOST="147.182.189.21"
REMOTE_DIR="/root/broker/secrets"
LOCAL_DIR="./secrets"

# Create local directory if it doesn't exist
mkdir -p "$LOCAL_DIR"

echo "Downloading files from $REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR to $LOCAL_DIR..."

scp "$REMOTE_USER@$REMOTE_HOST:$REMOTE_DIR/*" "$LOCAL_DIR/"

if [ $? -eq 0 ]; then
  echo "✅ Download complete."
else
  echo "❌ Failed to download files."
fi