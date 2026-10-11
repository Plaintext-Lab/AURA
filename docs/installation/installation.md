---
layout: default
title: "Installation"
nav_order: 3
description: "Installation instructions for aura."
permalink: /installation
has_children: true
has_toc: false
---

# Installation Types

- [Dockerfile Installation](/AURA/installation/docker-file)
- [Docker Compose Installation](/AURA/installation/docker-compose)
- [Unraid Installation](/AURA/installation/unraid)

## First Start

The first time aura starts (or whenever its configuration needs fixing), it opens the Onboarding page.

- If authentication is not set up yet, the setup page asks for a **setup code**. aura prints it in its logs, you can get this with a command like `docker logs aura`:
    ```
    aura is in setup mode. Enter this setup code on the setup page: K7QM-3RXA
    ```
    The code changes every time aura starts, and is only needed while setting up. It makes sure nobody else on your network can set up aura before you.
- If authentication is already set up, log in as usual to fix the configuration.

## Config Folder

aura stores its database (`AURA.db`, which also holds the configuration) and logs in the `/config` folder. See [Configuration](/AURA/config) for the environment variables that change these locations.

- The `/config` folder must be on a local filesystem. A local Docker volume or bind mount is fine (including on a NAS), but a network share (NFS/SMB) is not supported, as the database relies on file locking and shared memory that network shares do not reliably provide.
- While aura is running, `AURA.db-wal` and `AURA.db-shm` files will appear next to `AURA.db`. These are part of the database. To back up the database manually, stop aura first, or copy all three files together.
- aura also creates a backup of the database (e.g. `AURA_backup_v7_to_v8_<timestamp>.db`) before each database migration.
