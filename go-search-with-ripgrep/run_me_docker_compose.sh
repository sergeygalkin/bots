#!/bin/bash

set -e -o pipefail

docker compose version || { echo "Docker Compose not installed. Install it please";  exit 1; }

git checkout main
docker compose up --build -d
