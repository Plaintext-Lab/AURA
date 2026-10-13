import type { AppConfig } from "@/types/config/config";
import { DOWNLOAD_IMAGE_TYPE_OPTIONS } from "@/types/ui-options";

// Central default
export const defaultAppConfig = (): AppConfig =>
  ({
    auth: {
      enabled: false,
      password: "",
      oidc: {
        enabled: false,
        issuer_url: "",
        client_id: "",
        client_secret: "",
        redirect_url: "",
        allowed_emails: [],
        allowed_domains: [],
      },
      session_cookie_secure: "auto",
      trust_proxy_for_cookie_secure: false,
      allowed_origins: [],
    },
    logging: {
      level: "",
      file: "",
      keep_days: 7,
      keep_files: 7,
    },
    media_server: {
      type: "",
      url: "",
      api_token: "",
      libraries: [],
      user_id: "",
    },
    mediux: {
      api_token: "",
      download_quality: "",
    },
    jobs: {
      auto_download: { enabled: false, cron: "0 0 * * *" },
      refresh_media_items_and_collections: { enabled: true, cron: "0 4 * * *" },
      check_for_media_item_changes: { enabled: true, cron: "0 */6 * * *" },
      handle_temp_ignored_items: { enabled: true, cron: "0 */6 * * *" },
      refresh_mediux_users: { enabled: true, cron: "0 */12 * * *" },
      check_mediux_site_link: { enabled: true, cron: "0 */1 * * *" },
      backup_database: { enabled: true, cron: "0 3 * * *" },
    },
    backups: {
      keep: 7,
    },
    images: {
      cache_images: { enabled: false },
      save_images_locally: {
        enabled: false,
        path: "",
        episode_naming_convention: "",
      },
    },
    tmdb: {
      api_token: "",
    },
    labels_and_tags: {
      applications: [],
      remove_overlay_label_only_on_poster_download: false,
    },
    notifications: {
      enabled: false,
      providers: [],
      templates: {
        app_startup: {
          enabled: true,
          title: "Startup",
          message: "The application has started.",
          include_image: false,
        },
        test_notification: {
          enabled: true,
          title: "Test Notification",
          message: "This is a test notification.",
          include_image: false,
        },
        autodownload: {
          enabled: true,
          title: "Auto Download | {{ReasonTitle}}",
          message:
            "{{MediaItemTitle}}{{NewLine}}{{ImageName}}{{NewLine}}Set ID: {{SetID}}{{NewLine}}{{NewLine}}Reason:{{NewLine}}{{Reason}}",
          include_image: true,
        },
        download_queue: {
          enabled: true,
          title: "Download Queue | {{ReasonTitle}}",
          message:
            "{{MediaItemTitle}}{{NewLine}}{{ImageName}}{{NewLine}}Set ID: {{SetID}}{{NewLine}}{{NewLine}}Reason:{{NewLine}}{{Reason}}",
          include_image: true,
        },
        new_sets_available_for_ignored_items: {
          enabled: true,
          title: "New Sets Available for Ignored Item",
          message:
            "A new set has been detected for the previously ignored item '{{MediaItemTitle}}'. It is now a part of {{SetCount}} sets in MediUX, and will no longer be ignored in {{AppName}}.",
          include_image: true,
        },
        check_for_media_item_changes_job: {
          enabled: true,
          title: "Check For Media Item Changes Job",
          message:
            "The media item '{{MediaItemTitle}}' (TMDB ID: {{MediaItemTMDBID}}) in library '{{MediaItemLibraryTitle}}' could not be found in the media server cache.{{NewLine}}Reason:{{NewLine}}{{Reason}}{{NewLine}}{{NewLine}}{{Action}}{{NewLine}}{{MoreInfo}}",
          include_image: false,
        },
        sonarr_notification: {
          enabled: true,
          title: "Sonarr Notification | {{ReasonTitle}}",
          message:
            "{{MediaItemTitle}}{{NewLine}}{{ImageName}}{{NewLine}}Set ID: {{SetID}}{{NewLine}}Reason:{{NewLine}}{{Reason}}{{NewLine}}{{Result}}",
          include_image: true,
        },
      },
    },
    sonarr_radarr: {
      applications: [{ type: "", library: "", url: "", api_token: "" }],
    },
    user_preferences: {
      download_defaults: DOWNLOAD_IMAGE_TYPE_OPTIONS.map((option) => option.value),
      show_only_download_defaults: false,
      show_date_modified: false,
      default_add_to_queue: false,
    },
  }) satisfies AppConfig;
