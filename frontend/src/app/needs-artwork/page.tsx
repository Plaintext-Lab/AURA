"use client";

import { ReturnErrorMessage } from "@/services/api-error-return";
import { getArtworkSuggestions } from "@/services/suggestions/get-artwork-suggestions";
import { AlertCircle, Database, ExternalLink, RefreshCcw as RefreshIcon } from "lucide-react";
import { toast } from "sonner";

import React, { useCallback, useEffect, useRef, useState } from "react";

import { useRouter } from "next/navigation";

import { CustomPagination } from "@/components/shared/custom-pagination";
import { ErrorMessage } from "@/components/shared/error-message";
import Loader from "@/components/shared/loader";
import { RefreshButton } from "@/components/shared/refresh-button";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";

import { cn } from "@/lib/cn";
import { useMediaStore } from "@/lib/stores/global-store-media-store";

import type { APIResponse } from "@/types/api/api-response";
import type { ArtworkSuggestion, ArtworkSuggestionsResponse } from "@/types/suggestions/artwork-suggestions";

const ITEMS_PER_PAGE = 25;

const GAP_TYPE_OPTIONS = [
  { value: "", label: "Any gap" },
  { value: "no_set", label: "No saved set" },
  { value: "no_poster", label: "No poster" },
  { value: "no_backdrop", label: "No backdrop" },
  { value: "missing_season_posters", label: "Missing season posters" },
  { value: "missing_title_cards", label: "Missing title cards" },
];

const MEDIA_TYPE_OPTIONS = [
  { value: "", label: "All types" },
  { value: "movie", label: "Movies" },
  { value: "show", label: "Shows" },
];

const SORT_OPTIONS = [
  { value: "title", label: "Title" },
  { value: "year", label: "Year" },
  { value: "library", label: "Library" },
];

const REASON_CODE_LABELS: Record<string, string> = {
  no_set: "No Set",
  no_poster: "No Poster",
  no_backdrop: "No Backdrop",
  missing_season_posters: "Season Posters",
  missing_title_cards: "Title Cards",
};

const REASON_CODE_COLORS: Record<string, string> = {
  no_set: "bg-red-900/30 text-red-400 border-red-800",
  no_poster: "bg-orange-900/30 text-orange-400 border-orange-800",
  no_backdrop: "bg-yellow-900/30 text-yellow-400 border-yellow-800",
  missing_season_posters: "bg-blue-900/30 text-blue-400 border-blue-800",
  missing_title_cards: "bg-purple-900/30 text-purple-400 border-purple-800",
};

const NeedsArtworkPage: React.FC = () => {
  const router = useRouter();
  const { setMediaItem } = useMediaStore();

  const [suggestions, setSuggestions] = useState<ArtworkSuggestion[]>([]);
  const [totalItems, setTotalItems] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<APIResponse<unknown> | null>(null);

  const [currentPage, setCurrentPage] = useState(1);
  const [mediaType, setMediaType] = useState("");
  const [gapType, setGapType] = useState("");
  const [sortOption, setSortOption] = useState("title");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");

  const isFetchingRef = useRef(false);

  const fetchSuggestions = useCallback(async () => {
    if (isFetchingRef.current) return;
    isFetchingRef.current = true;
    setLoading(true);
    setError(null);

    const response = await getArtworkSuggestions({
      media_type: mediaType,
      gap_type: gapType,
      items_per_page: ITEMS_PER_PAGE,
      page_number: currentPage,
      sort_option: sortOption,
      sort_order: sortOrder,
    });

    if (response.status === "error" || !response.data) {
      setError(response);
      toast.error("Failed to load artwork suggestions");
    } else {
      setSuggestions(response.data.items ?? []);
      setTotalItems(response.data.total_items ?? 0);
    }

    setLoading(false);
    isFetchingRef.current = false;
  }, [currentPage, mediaType, gapType, sortOption, sortOrder]);

  useEffect(() => {
    fetchSuggestions();
  }, [fetchSuggestions]);

  // Reset to page 1 when filters change
  const handleFilterChange = (setter: (v: string) => void, value: string) => {
    setCurrentPage(1);
    setter(value);
  };

  const totalPages = Math.max(1, Math.ceil(totalItems / ITEMS_PER_PAGE));

  const handleOpenItem = (item: ArtworkSuggestion) => {
    // Build a minimal MediaItem shape to navigate to the existing media-item page
    setMediaItem({
      tmdb_id: item.tmdb_id,
      library_title: item.library_title,
      edition: item.edition,
      rating_key: item.rating_key,
      type: item.type,
      title: item.title,
      year: item.year,
      db_saved_sets: [],
      ignored_in_db: false,
      ignored_mode: "",
      has_mediux_sets: item.has_mediux_sets,
      updated_at: 0,
      added_at: 0,
      released_at: 0,
      latest_episode_added_at: 0,
      guids: [],
      content_rating: "",
      summary: "",
    });
    router.push("/media-item/");
  };

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex flex-col gap-2 mb-6">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <div>
            <h1 className="text-2xl font-bold">Needs Artwork</h1>
            <p className="text-muted-foreground text-sm mt-1">
              Media with AURA-managed artwork coverage gaps. Open an item to browse and select a set.
            </p>
          </div>
          <RefreshButton onClick={fetchSuggestions} isLoading={loading} />
        </div>

        {/* Filters */}
        <div className="flex flex-wrap gap-3 items-end mt-2">
          <div className="flex flex-col gap-1">
            <Label className="text-xs">Media type</Label>
            <Select value={mediaType} onValueChange={(v) => handleFilterChange(setMediaType, v)}>
              <SelectTrigger className="w-36 h-8 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {MEDIA_TYPE_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value} className="text-xs">
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1">
            <Label className="text-xs">Gap type</Label>
            <Select value={gapType} onValueChange={(v) => handleFilterChange(setGapType, v)}>
              <SelectTrigger className="w-48 h-8 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {GAP_TYPE_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value} className="text-xs">
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1">
            <Label className="text-xs">Sort by</Label>
            <Select value={sortOption} onValueChange={(v) => handleFilterChange(setSortOption, v)}>
              <SelectTrigger className="w-32 h-8 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {SORT_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value} className="text-xs">
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1">
            <Label className="text-xs">Order</Label>
            <Select value={sortOrder} onValueChange={(v) => handleFilterChange(setSortOrder, v as "asc" | "desc")}>
              <SelectTrigger className="w-24 h-8 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="asc" className="text-xs">
                  A → Z
                </SelectItem>
                <SelectItem value="desc" className="text-xs">
                  Z → A
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <Separator className="mt-3" />

        <p className="text-xs text-muted-foreground">
          {loading ? "Loading…" : `${totalItems} item${totalItems !== 1 ? "s" : ""} with coverage gaps`}
        </p>
      </div>

      {/* Body */}
      {loading && <Loader />}
      {!loading && error && <ErrorMessage response={error} />}
      {!loading && !error && suggestions.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-3 py-16 text-muted-foreground">
          <AlertCircle size={40} />
          <p className="text-lg font-medium">No coverage gaps found</p>
          <p className="text-sm">All tracked media appears to be fully managed by AURA.</p>
        </div>
      )}

      {!loading && !error && suggestions.length > 0 && (
        <div className="flex flex-col gap-3">
          {suggestions.map((item) => (
            <SuggestionCard key={`${item.tmdb_id}-${item.library_title}-${item.edition}`} item={item} onOpen={handleOpenItem} />
          ))}

          <CustomPagination
            currentPage={currentPage}
            totalPages={totalPages}
            setCurrentPage={setCurrentPage}
            itemsPerPage={ITEMS_PER_PAGE}
          />
        </div>
      )}
    </div>
  );
};

interface SuggestionCardProps {
  item: ArtworkSuggestion;
  onOpen: (item: ArtworkSuggestion) => void;
}

const SuggestionCard: React.FC<SuggestionCardProps> = ({ item, onOpen }) => {
  return (
    <Card className="border border-1 hover:shadow-lg transition-shadow">
      <CardHeader className="pb-2 pt-4 px-4">
        <div className="flex items-start justify-between gap-2 flex-wrap">
          <div className="flex flex-col">
            <CardTitle className="text-base font-semibold leading-tight">
              {item.title}
              {item.year > 0 && <span className="ml-2 text-muted-foreground font-normal text-sm">({item.year})</span>}
              {item.edition && (
                <Badge variant="secondary" className="ml-2 text-xs">
                  {item.edition}
                </Badge>
              )}
            </CardTitle>
            <p className="text-xs text-muted-foreground mt-0.5">
              {item.library_title} · {item.type === "show" ? "Show" : "Movie"}
            </p>
          </div>

          <div className="flex items-center gap-2 flex-shrink-0">
            {item.has_mediux_sets && (
              <Badge variant="outline" className="text-xs border-green-700 text-green-400">
                MediUX sets available
              </Badge>
            )}
            {item.has_saved_set && (
              <div className="rounded-full p-1 border border-blue-800" title="Has saved set">
                <Database className="text-blue-500" size={14} />
              </div>
            )}
            <Button size="sm" variant="outline" onClick={() => onOpen(item)} className="text-xs h-7 gap-1">
              <ExternalLink size={12} />
              Open
            </Button>
          </div>
        </div>
      </CardHeader>

      <CardContent className="px-4 pb-4">
        {/* Reason codes */}
        <div className="flex flex-wrap gap-1.5 mb-3">
          {item.reason_codes.map((code) => (
            <span
              key={code}
              className={cn("text-xs px-2 py-0.5 rounded border font-medium", REASON_CODE_COLORS[code] ?? "bg-muted text-muted-foreground border-muted")}
            >
              {REASON_CODE_LABELS[code] ?? code}
            </span>
          ))}
        </div>

        {/* Human-readable explanation */}
        <p className="text-xs text-muted-foreground mb-2">{item.reason_text}</p>

        {/* Coverage detail for shows */}
        {item.type === "show" && (item.season_poster_total > 0 || item.title_card_total > 0) && (
          <div className="flex flex-wrap gap-4 text-xs text-muted-foreground mt-1">
            {item.season_poster_total > 0 && (
              <span>
                Season posters:{" "}
                <span className={cn(item.season_poster_count < item.season_poster_total ? "text-orange-400" : "text-green-400")}>
                  {item.season_poster_count}/{item.season_poster_total}
                </span>
              </span>
            )}
            {item.title_card_total > 0 && (
              <span>
                Title cards:{" "}
                <span className={cn(item.title_card_count < item.title_card_total ? "text-orange-400" : "text-green-400")}>
                  {item.title_card_count}/{item.title_card_total}
                </span>
              </span>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default NeedsArtworkPage;
