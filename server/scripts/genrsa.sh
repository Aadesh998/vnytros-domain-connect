#!/bin/bash
# Generates the Domain Connect keypair into keys/. Run via `make keys`.
# keys/ is gitignored: never commit these files.
set -euo pipefail

mkdir -p keys
openssl genrsa -out keys/private_key.pem 4096
openssl rsa -in keys/private_key.pem -pubout -out keys/public_key.pem
chmod 600 keys/private_key.pem
