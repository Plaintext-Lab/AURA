export interface MediaActivitySummary {
  id: number;
  source: string;
  library_id: string;
  rating_key: string;
  grandparent_key: string;
  media_type: "movie" | "show" | "episode";
  tmdb_id: string;
  tvdb_id: string;
  title: string;
  play_count: number;
  watch_time_secs: number;
  last_watched: string;
  window_start: string;
  window_end: string;
  synced_at: string;
}

export interface ActivitySyncStatus {
  last_success_at: string;
  last_error_at: string;
  last_error: string;
  is_stale: boolean;
}

export interface ActivitySummariesResponse {
  summaries: MediaActivitySummary[];
  status: ActivitySyncStatus;
}
