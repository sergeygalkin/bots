#!/bin/bash

set -e -o pipefail

go version || { echo "Go not installed. Please install version >= 1.22.5";  exit 1; }
ansible-playbook --version || { echo "Ansible not installed. Please install >= 2.17";  exit 1; }

go mod download && go mod verify
go build -v
ansible-playbook -c local -i localhost, deploy.yaml
