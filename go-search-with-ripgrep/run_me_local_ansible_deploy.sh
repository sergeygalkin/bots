#!/bin/bash

set

go version || { echo "Go not installed. Install please.";  exit 1; }
ansible-playbook --version || { echo "Ansible not installed. Install please.";  exit 1; }

go mod download && go mod verify
go build -v
ansible-playbook -c local -i localhost, deploy.yaml
