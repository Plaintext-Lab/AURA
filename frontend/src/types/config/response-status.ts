import type { AppConfig } from "@/types/config/config";

export interface AppStatusResponse {
  config_loaded: boolean;
  config_valid: boolean;
  needs_setup: boolean;
  current_setup: AppConfig;
  media_server_name?: string;
  mediux_site_link?: string;
  app_fully_loaded: boolean;
  app_version: string;
  app_loading_step: string;
  api_key_configured: boolean;
  auth_disabled_by_env?: boolean; // AURA_DISABLE_AUTH has turned off authentication
}
