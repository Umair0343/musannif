# Musannif

Server for Musannif, A sophisticated collaborative Markdown editor designed for teams, enabling seamless real-time collaboration and creativity.

## Architecture

### High-Level Diagram

![system-architecture-diagram](.github/assets/system-architecture.drawio.png)

### Core Ideas

- Clients communicate diffs back and forth through a central server that resolves conflicts should any arise
- If a central server is reaching computational limits, it informs a master server to spin up a new server to handle the load, and transmits data to it, as well as transferring clients to it
- CHECK: Move database to a separate server that communicates with the master and/or worker servers

## Installing

- The easiest way to install Musannif is to simply grab an automatically created release

### Build locally

```bash
git clone https://github.com/musannif-md/musannif.git
cd musannif
cp config_example.yaml config.yaml # and modify the new file accordingly
make
```

## Usage

```bash
./musannif --signup -username <username> -password <password> # Optional
./musannif -serve
```

## TODO

- [x] Init a basic logger - create wrapper over zerolog
- [x] Mete out system architecture - how to approach horizontal scaling?
- [x] CI/CD: Publish new 'release' whenever a tagged commit is pushed
- [x] File/'Note' management on disk
- [x] Core websocket architecture
- [x] Ping-pong
- [x] Extract username from JWT as opposed to relying on JSON body
- [ ] Diff algorithm
    - [ ] Single user note modification
    - [ ] Concurrency: real-time collaboration w/ multiple users
- [ ] **Bug**: note name shouldn't contain extension whilst being stored/passed
- [x] Disconnect other clients if the host disconnects
- [ ] Note sharing via URL
- [ ] Fix Dockerfile, add persistent storage + networking support (through Docker Compose?) & configure CI/CD for pushing image to DockerHub
- [ ] User directory/Team management
- [ ] Shift to Protobufs
- [ ] 'Recently Deleted' note section
- [ ] ???
