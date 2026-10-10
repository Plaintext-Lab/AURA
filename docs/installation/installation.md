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

-   [Dockerfile Installation](/AURA/installation/docker-file)
-   [Docker Compose Installation](/AURA/installation/docker-compose)
-   [Unraid Installation](/AURA/installation/unraid)

## Config Folder

aura stores its configuration, logs and database (`AURA.db`) in the `/config` folder.

-   The `/config` folder must be on a local filesystem. A local Docker volume or bind mount is fine (including on a NAS), but a network share (NFS/SMB) is not supported, as the database relies on file locking and shared memory that network shares do not reliably provide.
-   While aura is running, `AURA.db-wal` and `AURA.db-shm` files will appear next to `AURA.db`. These are part of the database. To back up the database manually, stop aura first, or copy all three files together.
-   aura also creates a backup of the database (e.g. `AURA_backup_v7_to_v8_<timestamp>.db`) before each database migration.
