import type { SelectedTypes } from "@/types/media-and-posters/media-item-and-library";

export interface LibraryGroup {
  id: string;
  name: string;
  media_type: "movie" | "show";
  library_ids: string[];
  created_at: string;
  updated_at: string;
}

export interface LibraryGroupPolicy {
  id: string;
  group_id: string;
  tmdb_id: string;
  edition: string;
  set_id: string;
  selected_types: SelectedTypes;
  auto_download: boolean;
  future_updates_only: boolean;
  last_reconciled?: string;
  reconcile_status: string;
  created_at: string;
  updated_at: string;
}

export interface LibraryItemSets {
  library_id: string;
  library_title: string;
  saved_sets: Array<{ id: string; user_created: string; selected_types: SelectedTypes }>;
}

export interface LibraryGroupPreviewItem {
  tmdb_id: string;
  edition: string;
  title: string;
  year: number;
  media_type: "movie" | "show";
  library_sets: LibraryItemSets[];
  has_conflict: boolean;
}

export interface LibraryGroupPreview {
  group_id?: string;
  match_count: number;
  conflict_count: number;
  items: LibraryGroupPreviewItem[];
}

export interface ReconcileDecision {
  set_id: string;
  selected_types: SelectedTypes;
  auto_download: boolean;
}
