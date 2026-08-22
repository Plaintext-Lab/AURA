import apiClient from "@/services/api-client";
import { ReturnErrorMessage } from "@/services/api-error-return";

import { log } from "@/lib/logger";

import type { APIResponse } from "@/types/api/api-response";
import type { ArtworkSuggestionsParams, ArtworkSuggestionsResponse } from "@/types/suggestions/artwork-suggestions";

export const getArtworkSuggestions = async (
  params: ArtworkSuggestionsParams
): Promise<APIResponse<ArtworkSuggestionsResponse>> => {
  try {
    const response = await apiClient.get<APIResponse<ArtworkSuggestionsResponse>>(`/suggestions/artwork`, {
      params,
    });
    if (response.data.status === "error") {
      throw new Error(response.data.error?.message || "Unknown error fetching artwork suggestions");
    }
    log(
      "INFO",
      "API - Suggestions",
      "Get Artwork Suggestions",
      `Fetched ${response.data?.data?.items.length ?? 0} suggestions`
    );
    return response.data;
  } catch (error) {
    log(
      "ERROR",
      "API - Suggestions",
      "Get Artwork Suggestions",
      `Failed: ${error instanceof Error ? error.message : "Unknown error"}`,
      error
    );
    return ReturnErrorMessage<ArtworkSuggestionsResponse>(error);
  }
};
