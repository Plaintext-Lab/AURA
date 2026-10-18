---
layout: default
title: "Configuration"
nav_order: 2
description: "Configuration instructions for aura."
permalink: /config
---

# Configuration

aura stores its configuration in its database (`AURA.db` in the `/config` folder). You can set up the configuration during the onboarding process, and change it any time from the `Settings` page.

The options below use the `config.yaml` format, which you can still use to set up aura:

1. **Create a `config.yaml` File**:
    - Create a new file named `config.yaml`, and set the options you need. Be sure to replace all placeholder values (e.g., `YOUR_API_KEY_HERE`, `YOUR_SERVER_URL`) with your actual credentials and URLs.

2. **Place the `config.yaml` File**:
    - Place the file in the `/config` folder on your Docker container, and start aura.
    - On startup, aura imports the file into its database and renames it to `config.yaml.imported`.

Once the configuration is in the database, aura ignores any new `config.yaml` file. To import a `config.yaml` file again, replacing the current configuration, set the `AURA_IMPORT_CONFIG=true` environment variable and restart aura.

### Environment Variables

- **`CONFIG_PATH`**: The folder aura stores its database, logs and other files in. Defaults to `/config`.
- **`DATABASE_PATH`**: The path of the database file. A relative path is relative to `CONFIG_PATH`. Defaults to `AURA.db`.
    - Older versions of aura set this with `Database.Path` in `config.yaml`. If you changed it there, set `DATABASE_PATH` to the same value.
- **`AURA_IMPORT_CONFIG`**: Set to `true` to import a `config.yaml` file on startup, even if the configuration is already in the database.
- **`AURA_DISABLE_AUTH`**: Set to `true` to turn off authentication, even if it is enabled in the configuration. Use this if you are locked out (e.g. a forgotten password or a broken identity provider): log in, fix the Authentication settings, then remove the variable and restart aura. While it is set, anyone who can reach aura can use it, and aura logs a warning on every startup.

### Backing Up Your Configuration

In `Settings` → `Advanced`, **Export Configuration** downloads your configuration as a YAML file (the `config.yaml` format), and **Import Configuration** restores one. The exported file includes your passwords, tokens and API keys, so keep it private. Importing checks the configuration (including the Media Server and MediUX connections) before anything is saved. Your API key is not changed by an import.

> **Note:** The database and any `config.yaml` file contain sensitive information (e.g. tokens and API keys). Keep them secure and do not share them publicly.

---

# Configuration Options

## Authentication

Aura has two independent ways to authenticate against the API, plus an optional third login method for the browser:

- **Browser session (password or OIDC)** - logging in through the app's UI sets an HttpOnly session cookie. This is for interactive browser use only; there is no token returned to copy into a script.
- **API key** - a single key for programmatic/integration access (scripts, the Sonarr/Radarr webhook). Sent as the `X-Api-Key` header (or, for the Sonarr/Radarr webhook specifically, as the password in HTTP Basic Auth - see [Sonarr Webhook Integration](sonarr-webhook-integration) - since Sonarr/Radarr's built-in Webhook connection type has no custom-header support). Generate/regenerate it under `Settings` → `Authentication` → `API Key`; it's shown once and is never stored or retrievable in plaintext again. Regenerating immediately invalidates the previous key everywhere it's used.

- **Example**:

```yaml
Auth:
    Enabled: true
    Password: YOUR_ARGON2ID_HASH_HERE
    SessionCookieSecure: auto
    OIDC:
        Enabled: false
```

While this password authentication method is effective, it is important to keep your password secure and not share it with others. For enhanced security, consider using solutions like [Authentik](https://goauthentik.io/), [Authelia](https://www.authelia.com/), [Tinyauth](https://tinyauth.app/), or Aura's own built-in OIDC support (below).  
**I am not a security expert** 😅

### Enabled

- **Default**: `false`
- **Options**: `true` or `false`
- **Description**: Whether to enable authentication.
- **Details**: If set to `true`, you will be required to authenticate before accessing the application.

### Password

- **Default**: `null`
- **Options**: Any valid Argon2id hash
- **Description**: The password hash used to authenticate user.
- **Details**: This password is used to authenticate user when they log in to the application. It is recommended to use a strong, unique password for this purpose. You can generate a new Argon2id hash using tools like [Argon2 Online](https://argon2.online/). You can use the default settings.
  ![Argon2 Online](assets/argon2-online.png)
- **Note**: Replace `YOUR_ARGON2ID_HASH_HERE` with the actual Argon2id hash of your desired password.

### SessionCookieSecure

- **Default**: `auto`
- **Options**: `auto`, `always`, `never`
- **Description**: Controls the `Secure` attribute on the browser session cookie.
- **Details**: `auto` (recommended) sets `Secure` based on whether the incoming request actually used HTTPS - this is what most self-hosted/LAN setups (e.g. `http://192.168.1.50:3000`) need, since a hardcoded `Secure` cookie is silently rejected by the browser over plain HTTP and login will appear to fail with no clear error. Only override this if you know your deployment needs it.

### TrustProxyForCookieSecure

- **Default**: `false`
- **Options**: `true` or `false`
- **Description**: When `true`, also treats an `X-Forwarded-Proto: https` header as HTTPS for the `SessionCookieSecure: auto` check.
- **Details**: Use this if you run Aura behind a reverse proxy (Nginx, Traefik, etc.) that terminates TLS and forwards plain HTTP to Aura. Only enable this if your proxy is guaranteed to set/overwrite this header on the way in - otherwise it can be spoofed by a client that reaches Aura directly.

### AllowedOrigins

- **Default**: `[]` (none)
- **Description**: Extra origins allowed to make credentialed cross-origin requests to the API.
- **Details**: The standard deployment (the bundled Next.js server proxying `/api/*` requests to this backend) is same-origin from the browser's perspective and never needs this. Only set it if you're calling the API directly from a different origin than the one serving the frontend.

### OIDC

- **Default**: disabled
- **Description**: Optional Single Sign-On login via any standard OIDC identity provider (Authelia, Authentik, Pocket ID, Keycloak, etc.), in addition to (not instead of) the password above.
- **Example**:

```yaml
Auth:
    OIDC:
        Enabled: true
        IssuerURL: https://idp.example.com
        ClientID: aura
        ClientSecret: YOUR_OIDC_CLIENT_SECRET
        RedirectURL: https://aura.example.com/api/auth/oidc/callback
        AllowedEmails:
            - you@example.com
        AllowedDomains:
            - example.com
```

- **IssuerURL / ClientID / ClientSecret / RedirectURL**: standard OIDC client registration values from your identity provider. `RedirectURL` must be registered with the IdP exactly as configured here.
- **IssuerURL** is the base URL of your identity provider (e.g. `https://idp.example.com`), not its `/.well-known/openid-configuration` URL. If the discovery URL is entered, aura removes the `/.well-known/openid-configuration` part automatically.
- **RedirectURL** must be your aura URL followed by `/api/auth/oidc/callback`. If it points anywhere else, Single Sign-On stays unavailable and the reason is shown when saving the settings.
- **AllowedEmails / AllowedDomains**: optional allowlists. If **both** are left empty, **any** user who successfully authenticates with your identity provider is granted full access to Aura - there is no per-user account model, so this is equivalent to sharing the password with everyone who has an IdP account. The Settings UI shows a warning banner when OIDC is enabled with no allowlist configured.
- All of the above can be configured via `Settings` → `Authentication` in the UI instead of editing the YAML directly.

---

## Logging

- **Example**:

```yaml
Logging:
    Level: DEBUG
    KeepDays: 7
    KeepFiles: 7
```

### Level

- **Default**: `INFO`
- **Options**: `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`
- **Description**: The logging level for aura.
- **Details**:
    - `TRACE`: Most detailed logging, useful for debugging.
    - `DEBUG`: Less detailed than TRACE, but still provides useful information for debugging.
    - `INFO`: General information about the application's operation.
    - `WARN`: Indicates potential issues that are not necessarily errors.
    - `ERROR`: Indicates errors that occur during the application's operation.
- **Note**: The logging level can be adjusted based on your needs. For production environments, it is recommended to use `INFO` or `WARN` to reduce log verbosity. If you run into issues, you can temporarily set it to `DEBUG` or `TRACE` for more detailed logs.

### KeepDays

- **Default**: `7`
- **Options**: `1` to `365`
- **Description**: How many days old log files are kept before they are deleted.
- **Details**: aura starts a new log file when the current one reaches 25 MB. Old files are cleaned up at that point, so on a quiet install they can stay a little longer than `KeepDays`.

### KeepFiles

- **Default**: `7`
- **Options**: `1` to `100`
- **Description**: How many old log files are kept before the oldest are deleted.
- **Details**: Old log files are deleted when either limit is reached, `KeepDays` or `KeepFiles`.

---

## MediaServers

aura applies images to one **main** media server, and can also apply them to any number of other media servers. Set them up in **Settings > Media Servers**.

- The main media server is the one shown on the Home page. It is the only one that saves images locally and listens for Plex events.
- When an image is applied, it is applied on the main media server first, then on each other media server that has the same item. Items are matched by their TMDB ID, in the libraries that match the item's library on the main media server.
- If another media server has no matching edition of a movie, the image is applied to its base movie.
- Ratings are applied on every Plex and Silo media server, and labels on every Plex media server.
- Collection images are only applied on the main media server.
- A media server that can't be reached is skipped and logged as a warning. It doesn't stop aura or the main media server.
- Images are only applied to the other media servers as they are applied from then on. To apply the sets you saved before adding a media server, use **Apply Saved Sets** on it in **Settings > Media Servers**. It runs in the background, shows its progress, and can be stopped. The main media server can't be changed while it runs.

- **Example with a main Plex server and a Silo server**:

```yaml
MainMediaServer: a1b2c3d4e5f60718
MediaServers:
    - ID: a1b2c3d4e5f60718
      Name: Plex
      Enabled: true
      Type: Plex
      URL: YOUR_PLEX_SERVER_URL_HERE
      ApiToken: YOUR_PLEX_API_TOKEN_HERE
      Libraries:
          - Title: 4K Movies
          - Title: Movies
          - Title: Series
    - ID: 9f8e7d6c5b4a3921
      Name: Silo
      Enabled: true
      Type: Silo
      URL: YOUR_SILO_SERVER_URL_HERE
      ApiToken: YOUR_SILO_API_KEY_HERE
      Libraries:
          - Title: Films
          - Title: TV Shows
      LibraryMap:
          Films: Movies
          TV Shows: Series
```

- **Config files from older versions** have a single `MediaServer`. It can still be imported, and becomes the main media server:

```yaml
MediaServer:
    Type: Plex
    URL: YOUR_PLEX_SERVER_URL_HERE
    ApiToken: YOUR_PLEX_API_TOKEN_HERE
    Libraries:
        - Title: Movies
```

### MainMediaServer

- **Description**: The `ID` of the main media server.
- **Details**: Change the main media server with **Make Main** on a media server in **Settings > Media Servers**, not by editing this value. aura first shows what happens to your saved items, then moves them to the new main media server's items: their saved sets, ignored status and download history move with them. Saved items that aren't on the new main media server are kept, and marked as not on the server. Saved items that become the same item on the new main media server, like two editions of a movie on a media server without editions, are merged and keep the saved sets of both. The library maps change to the new main media server's libraries. The switch can't happen while downloads are in the queue.

### ID

- **Description**: A unique ID for the media server.
- **Details**: aura creates the ID when the media server is added. Keep it the same when editing a config file, as `MainMediaServer` uses it.

### Name

- **Default**: The media server type
- **Description**: The name shown for the media server in aura.

### Enabled

- **Default**: `true`
- **Options**: `true` or `false`
- **Description**: Whether aura applies images to this media server.
- **Details**: Turn a media server off to stop using it without removing it. The main media server is always enabled.

### LibraryMap

- **Description**: Which main media server library each library of this media server matches, by title.
- **Details**: Libraries with the same title as a main media server library match it automatically. An empty value means the library doesn't match any main media server library. Not used for the main media server.

### Type

- **Options**: `Plex`, `Emby`, `Jellyfin`, `Silo`
- **Description**: The type of media server you are using.
- **Details**: This option specifies the type of media server that aura will interact with. Depending on your choice, aura will use the appropriate API and methods to manage images and metadata.

### URL

- **Description**: The URL of the media server.
- **Details**: This option specifies the URL of the media server that aura will interact with.
- **Note**: Replace `YOUR_PLEX_SERVER_URL_HERE`, `YOUR_EMBY_SERVER_URL_HERE`, `YOUR_JELLYFIN_SERVER_URL_HERE`, or `YOUR_SILO_SERVER_URL_HERE` with the actual URL of your media server. Make sure to include the protocol (e.g., `http://` or `https://`) in the URL. For Silo, use the main Silo server URL, not the Jellyfin compatibility URL.
- **Example**: `http://localhost:32400`, `https://my-emby-server.com`, or `http://jellyfin.example.com`.

### ApiToken

- **Description**: The authentication token for the media server.
- **Details**: This option specifies the authentication token required to access the media server's API. You can obtain this token from your media server's settings or API documentation.
- **Note**: The token is necessary for aura to authenticate and perform actions on your media server. Make sure to keep this token secure and do not share it publicly.
- **Silo**: Create the API Key from an admin account, as applying images and refreshing metadata require admin access. Silo limits how many requests an API Key can make by default, and aura makes one request per item when loading your libraries. Setting the API Key's rate tier to `elevated` on the Silo server is recommended, especially for large libraries. If you notice that you are still hitting the rate limit, you can set it to a higher number in Silo (Admin > Settings > Security & Access, and then changing the Elevated API Key Rate Limit).

### Libraries

- **Description**: The title of the media server library to use.
- **Details**: This option specifies the title of the library on your media server that aura will interact with. aura will use this library to manage images and metadata.
- **Note**: Ensure that the library title matches exactly with the title on your media server, including case sensitivity. Only show and movies libraries are supported.

### EnableSortByEpisodeAddedDate (Plex Only)

- **Default**: `false`
- **Options**: `true` or `false`
- **Description**: Whether the option to Sort content by Episode Added Date should be enabled in the UI for Plex users. Only used for the main media server.
- **Details**: If set to `true`, users will have the option to sort their Plex library content by the date episodes were added. This can be useful for quickly identifying recently added content. If set to `false`, this sorting option will not be available in the UI. The recommended setting is `false` as it is very slow to sort by episode added date, especially for larger libraries.

### EnablePlexEventListener (Plex Only)

- **Default**: `false`
- **Options**: `true` or `false`
- **Description**: Whether to enable the Plex Event Listener for real-time updates for the "Refresh Metadata" action. Only used for the main media server.
- **Details**: If set to `true`, aura will listen for Plex events to trigger real-time updates when the "Refresh Metadata" action is performed. This allows for faster updates to your media library without waiting for the next scheduled update. If set to `false`, updates will only occur during the scheduled update process. Enabling this option may increase resource usage, so it is recommended to only enable it if you want real-time updates and have the resources to support it.

---

## Mediux

- **Example**:

```yaml
Mediux:
    ApiToken: YOUR_MEDIUX_API_TOKEN_HERE
    DownloadQuality: optimized
```

### ApiToken

- **Description**: The API token for MediUX.
- **Details**: This option specifies the API token required to access MediUX's API. This can be obtained by creating an account on [MediUX](https://mediux.io/) and generating an API token in your account settings.
- **Note**: This is not yet available to the public, but will be in the future.
  If you would like to test out aura, please join us on [Discord](https://discord.gg/YAKzwKPwyw) and make your way over to repo-chats > mediux-team/aura. Here you will find a pinned message that contains a shared token you can use for testing. Once the new MediUX site is live, you will be able to create your own tokens.

### DownloadQuality

- **Default**: `optimized`
- **Options**: `optimized`, `original`
- **Description**: The quality of images to download from MediUX.
- **Details**: This option specifies the quality of images to download from MediUX.
    - `optimized`: Downloads images that are optimized for space savings and performance.
    - `original`: Downloads the original images without any optimization.

---

## Jobs

Aura runs several background jobs on a schedule to keep your library, saved sets, and MediUX data in sync. Each job can be independently enabled/disabled and given its own cron schedule.

> **Tip**: You don't need to edit `config.yaml` for this. The **Jobs** page in the UI lets you toggle each job on/off, edit its cron schedule, and trigger a manual "Run Now" - changes are saved automatically. The reference below is for anyone who prefers to manage `config.yaml` directly (or is scripting a deployment).

- **Example**:

```yaml
Jobs:
    AutoDownload:
        Enabled: true
        Cron: "0 0 * * *"
    RefreshMediaItemsAndCollections:
        Enabled: true
        Cron: "0 4 * * *"
    CheckForMediaItemChanges:
        Enabled: true
        Cron: "0 */6 * * *"
    HandleTempIgnoredItems:
        Enabled: true
        Cron: "0 */6 * * *"
    RefreshMediuxUsers:
        Enabled: true
        Cron: "0 */12 * * *"
    CheckMediuxSiteLink:
        Enabled: true
        Cron: "0 */1 * * *"
    BackupDatabase:
        Enabled: true
        Cron: "0 3 * * *"
```

Every job entry has the same two fields:

- **Enabled**: `true` or `false`. When `false`, the job will not run automatically on its schedule, but can still be triggered manually via "Run Now" on the Jobs page. If omitted entirely, the job falls back to its own built-in default below rather than being treated as disabled - so it's safe to only list the jobs you actually want to customize.
- **Cron**: A standard 5-field cron expression controlling how often the job runs. If omitted, the job's built-in default schedule (below) is used. Use a site like [crontab.guru](https://crontab.guru/) to help you build and validate cron expressions.

### AutoDownload

- **Default**: `Enabled: true`, `Cron: "0 0 * * *"` (daily at midnight)
- **Description**: Automatically checks your saved sets for new or updated images and downloads them.
- **Note**: Enabling this job may result in increased network usage, as aura will periodically check for updates and download new images.

### RefreshMediaItemsAndCollections

- **Default**: `Enabled: true`, `Cron: "0 4 * * *"` (daily at 4:00 AM)
- **Description**: Re-scans your Media Server libraries, pulling in new and updated movies, shows, and collections to update the backend cache.

### CheckForMediaItemChanges

- **Default**: `Enabled: true`, `Cron: "0 */6 * * *"` (every 6 hours)
- **Description**: Compares Media Items from the database with the Media Server cache to see if any items have been removed or changed. If a Media Item is no longer in the cache, it will be removed from the database unless it has Saved Sets or is Temp Ignored.

### HandleTempIgnoredItems

- **Default**: `Enabled: true`, `Cron: "0 */6 * * *"` (every 6 hours)
- **Description**: Rechecks temporarily ignored media items to see if a matching set has since become available on MediUX.

### RefreshMediuxUsers

- **Default**: `Enabled: true`, `Cron: "0 */12 * * *"` (every 12 hours)
- **Description**: Updates the list of MediUX users in the backend cache, so that search results reflect the current set of users within MediUX.

### CheckMediuxSiteLink

- **Default**: `Enabled: true`, `Cron: "0 */1 * * *"` (hourly)
- **Description**: MediUX currently has 2 sites: one for live and one for testing. This job checks the availability of the testing site, and falls back to the live site if the testing site is unavailable.

### BackupDatabase

- **Default**: `Enabled: true`, `Cron: "0 3 * * *"` (daily at 3:00 AM)
- **Description**: Saves a copy of the database, with your settings and saved sets, to the `backups` folder next to it. The oldest copies are deleted so only the number set in [Backups](#backups) are kept.

---

## Backups

aura backs up its database with the [BackupDatabase](#backupdatabase) job. You can also back it up any time with **Back Up Now** in **Settings > Backups**, which lists the backups too.

- **Example**:

```yaml
Backups:
    Keep: 7
```

### Keep

- **Default**: `7`
- **Options**: `1` to `100`
- **Description**: How many backups are kept. The oldest are deleted when a new backup is made.

### Where Backups Are Saved

Backups are saved in a `backups` folder next to the database, so `/config/backups` by default, as `AURA_<date>_<time>.db`. Each one is a complete copy of the database.

To restore a backup:

1. Stop aura.
2. Move `AURA.db` out of the `/config` folder, along with `AURA.db-wal` and `AURA.db-shm` if they are there.
3. Copy the backup into the `/config` folder and rename it to `AURA.db`.
4. Start aura.

---

## Images

- **Example**:

```yaml
Images:
    CacheImages:
        Enabled: false
    SaveImagesLocally:
        Enabled: false
        Path: ""
        RunningOnWindows: false
        NameTemplates:
            Poster: "poster"
            Backdrop: "backdrop"
            SeasonPoster: "season{{SeasonNumber}}-poster"
            SpecialSeasonPoster: "season-specials-poster"
            Titlecard: "{{EpisodeFileName}}-thumb"
```

## CacheImages.Enabled

- **Default**: `false`
- **Options**: `true` or `false`
- **Description**: Whether to cache images locally.
- **Details**: If set to `true`, aura will cache images to reduce load times and improve performance. This is particularly useful for frequently accessed images.Keep in mind that enabling this option will increase disk space usage as images will be stored locally.

## SaveImagesLocally.Enabled

- **Default:** `false`
- **Options:** `true` or `false`
- **Description:** Whether to save images locally.
- **Details:**
    - If `true`, images are applied on the Media Server and a copy is saved in the same directory as the Media Server content.
    - If `false`, images are applied on the Media Server but not saved next to the content.
    - This option works the same for **Plex**, **Emby**, **Jellyfin** and **Silo**.
    - For **Silo**, the saved copy does not change what Silo shows, as Silo only reads these files when it first scans an item.
    - aura must be able to reach the media folders at the same paths the Media Server uses (e.g. through docker volume mounts).

## SaveImagesLocally.Path

- **Default:** `""` (empty string)
- **Options:** Any valid file path
- **Description:** The path where images should be saved if `SaveImagesLocally.Enabled` is `true`.
- **Details:**
    - If set to a valid path, images will be saved to that directory.
    - If left empty, images will be saved next to the media content.
    - Ensure the specified path is added to your docker volume mounts.
    - Ensure the specified path is writable by the application.

## SaveImagesLocally.NameTemplates

- **Default:** the standard names above, which work with Plex, Emby, Jellyfin and Silo.
- **Description:** The file name of each saved image type (`Poster`, `Backdrop`, `SeasonPoster`, `SpecialSeasonPoster` and `Titlecard`), without the extension.
- **Details:**
    - Variables: `{{Title}}`, `{{Year}}`, `{{TMDBID}}`, `{{LibraryTitle}}`, `{{LibraryID}}`, `{{Edition}}`, `{{SeasonNumber}}`, `{{EpisodeNumber}}` and `{{EpisodeFileName}}`.
    - The extension is added for you. Use `/` for folders. A name that starts with `/` is placed from `SaveImagesLocally.Path` instead of next to the media.
    - An empty name, or one with an unknown variable or `..`, uses the standard name.
    - Example for Agregarr's base posters, with `Path` set to that folder: `/{{LibraryTitle}}-{{LibraryID}}/{{Title}} ({{Year}}) tmdb-{{TMDBID}}/poster`.
    - This replaces `EpisodeNamingConvention`. A config with `"static"` gets the title card name `S{{SeasonNumber}}E{{EpisodeNumber}}`.

## SaveImagesLocally.RunningOnWindows

- **Default:** `false`
- **Options:** `true` or `false`
- **Description:** Whether the application is running on Windows when saving images locally.
- **Details:**
    - If `true`, file paths will use Windows-style backslashes (`\`) and handle file permissions accordingly.
    - If `false`, file paths will use Unix-style forward slashes (`/`) and handle file permissions for Unix-based systems.
- **Note:** This option is only applicable when `SaveImagesLocally.Enabled` is `true`. It helps ensure that file paths and permissions are correctly handled based on the operating system you are running the application on.

---

## Labels and Tags

Aura supports adding and removing labels (tags) on Plex items after processing. This is useful for organizing your media library, marking items for automation, or integrating with other tools.

- **Example**:

```yaml
LabelsAndTags:
    RemoveOverlayLabelOnlyOnPosterDownload: false
    Applications:
        - Application: Plex
          Enabled: true
          Add:
              - "Overlay"
              - "4K"
          Remove:
              - "OldLabel"
          AddLabelTagForSelectedTypes: true
```

### Applications

- **Description**:  
  An array of label/tag configuration blocks, one per supported application.
- **Fields**:
    - `Application`: The name of the application (e.g., `Plex`, `Sonarr` or `Radarr`).
    - `Enabled`: Set to `true` to enable label/tag management for this application.
    - `Add`: A list of labels/tags to add to items after processing.
    - `Remove`: A list of labels/tags to remove from items after processing.
    - `AddLabelTagForSelectedTypes`: A boolean to add labels in Plex and tags in Sonarr/Radarr for each selected type (e.g., aura-poster, aura-backdrop).

    ## RemoveOverlayLabelOnlyOnPosterDownload

- **Default**: `false`
- **Options**: `true` or `false`
- **Description**: Whether to only remove the "Overlay" label when a poster is downloaded, and not when a season poster or titlecard is downloaded.
- **Details**: If set to `true`, aura will only remove the "Overlay" label from Plex items when a poster is downloaded. If a season poster or titlecard is downloaded, the "Overlay" label will not be removed. This allows you to keep the "Overlay" label on items that have season posters or titlecards, while only removing it from items that have posters downloaded. If set to `false`, the "Overlay" label will be removed whenever any type of image (poster, season poster, or titlecard) is downloaded.

#### Example Use Case

If you want Aura to add the labels `Overlay` and `aura` to your Plex items, and remove the label `OldLabel`, your config would look like:

```yaml
LabelsAndTags:
    Applications:
        - Application: Plex
          Enabled: true
          Add:
              - "Overlay"
              - "aura"
          Remove:
              - "OldLabel"
          AddLabelTagForSelectedTypes: true
        - Application: Sonarr
          Enabled: true
          Add:
              - aura
          Remove:
              - "some-old-label"
          AddLabelTagForSelectedTypes: true
        - Application: Radarr
          Enabled: true
          Add:
              - aura
          Remove:
              - "some-old-label"
```

#### Notes

- You can leave `Add` or `Remove` empty if you only want to add or only want to remove labels.
- Only applications with `Enabled: true` will be processed.
- This structure is extensible for future support of other applications (such as Sonarr or Radarr).
- Any tags for Sonarr/Radarr have to be in lowercase.

---

## Notifications

Configure one or more providers. Notifications can be disabled globally or per provider.

Example:

```yaml
Notifications:
    Enabled: true # Master switch (false = ignore all providers)
    Providers:
        - Provider: "Discord"
          Enabled: true
          Discord:
              Webhook: YOUR_DISCORD_WEBHOOK_URL
        - Provider: "Pushover"
          Enabled: true
          Pushover:
              ApiToken: YOUR_PUSHOVER_APP_TOKEN
              UserKey: YOUR_PUSHOVER_USER_KEY
        - Provider: "Gotify"
          Enabled: true
          Gotify:
              URL: YOUR_GOTIFY_SERVER_URL
              ApiToken: YOUR_GOTIFY_APP_TOKEN
        - Provider: "Webhook"
          Enabled: true
          Webhook:
              URL: YOUR_WEBHOOK_URL
              Headers:
                  Some-Header: "HeaderValue"
                  Another-Header: "AnotherValue"
```

### Structure

- Notifications.Enabled  
  Global on/off. If false, Providers are loaded but not used.
- Notifications.Providers[]  
  Array of provider entries.

### Provider Entry Fields

| Field             | Required                               | Notes                                        |
| ----------------- | -------------------------------------- | -------------------------------------------- |
| Provider          | yes                                    | Case-sensitive. Supported: Discord, Pushover |
| Enabled           | yes                                    | If false, entry kept but skipped             |
| Discord.Webhook   | yes (when Provider=Discord & Enabled)  | Full Discord webhook URL                     |
| Pushover.ApiToken | yes (when Provider=Pushover & Enabled) | Your app token                               |
| Pushover.UserKey  | yes (when Provider=Pushover & Enabled) | Your user key                                |
| Gotify.URL        | yes (when Provider=Gotify & Enabled)   | Base URL for your Gotify server              |
| Gotify.ApiToken   | yes (when Provider=Gotify & Enabled)   | Your Gotify app token                        |

**Note**: Replace any `YOUR_...` placeholders with your actual configuration values. For URL fields, ensure you include the full URL with the appropriate protocol (e.g., `http://` or `https://`).

---

## Sonarr and Radarr Integration

- **Example**:

```yaml
SonarrRadarr:
    Applications:
        - Type: Sonarr
          Library: Series
          URL: YOUR_SONARR_URL
          ApiToken: YOUR_SONARR_API_TOKEN
        - Type: Sonarr
          Library: 4K Series
          URL: YOUR_SONARR_URL
          ApiToken: YOUR_SONARR_API_TOKEN
        - Type: Radarr
          Library: Movies
          URL: YOUR_RADARR_URL
          ApiToken: YOUR_RADARR_API_TOKEN
        - Type: Radarr
          Library: 4K Movies
          URL: YOUR_RADARR_URL
          ApiToken: YOUR_RADARR_API_TOKEN
```

Aura can interact with Sonarr and Radarr to add tags to your Sonarr/Radarr items after processing. This is useful for organizing your media library, marking items for automation, or integrating with other tools.

> **Note**: aura also supports a custom webhook integration from Sonarr to aura to redownload titlecards when an episode file is upgraded. View the [documentation](https://mediux-team.github.io/AURA/sonarr-webhook-integration) for more details.

### Type

- **Options**: `Sonarr`, `Radarr`
- **Description**: The type of application you are configuring.
- **Details**: This option specifies whether you are configuring Sonarr or Radarr. Depending on your choice, aura will use the appropriate API and methods to manage tags.

### URL

- **Description**: The URL of the Sonarr or Radarr server.
- **Details**: This option specifies the URL of the Sonarr or Radarr server that aura will interact with.
- **Note**: Replace `YOUR_SONARR_URL` or `YOUR_RADARR_URL` with the actual URL of your Sonarr or Radarr server. Make sure to include the protocol (e.g., `http://` or `https://`) in the URL.
- **Example**: `http://localhost:8989`, `https://my-sonarr-server.com`, or `http://my-radarr-server.com`.

### ApiToken

- **Description**: The API key for the Sonarr or Radarr server.
- **Details**: This option specifies the API key required to access the Sonarr or Radarr server's API. You can obtain this key from your Sonarr or Radarr server's settings.
- **Note**: The API key is necessary for aura to authenticate and perform actions on your Sonarr or Radarr server. Make sure to keep this key secure and do not share it publicly.

---
