# Bot for telegram on go

This is very stupid bot. Run ripgrep on your server and return result to chat.
On the Intel(R) Pentium(R) Silver J5040 CPU @ 2.00GHz, it takes about 1 minute per one request
for 2155 CSV files with 29G summary size. Disk is SSD.

## Run with docker compose (recommended)

### Preparation

1. install docker and docker-compose
2. `cp config.example.yaml config.yaml` and edit config file
3. `cp .env.example .env` and edit env file, change to real files paths

### Run

```
./run_me_docker_compose.sh
```

## Run as systemd service (do not have uninstall, not fully tested)

1. install golang >= 1.22 and ansible >= 2.17
2. `cp config.example.yaml config.yaml` and edit config file


### Run

```
./run_me_local_ansible_deploy.sh
```
