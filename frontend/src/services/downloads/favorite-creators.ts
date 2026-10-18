import apiClient from "@/services/api-client";
import { ReturnErrorMessage } from "@/services/api-error-return";

import { log } from "@/lib/logger";

import type { APIResponse } from "@/types/api/api-response";

export interface FavoriteCreatorsResult {
  checked: number; // Items without a saved set
  queued: number; // Items a set was added to the Download Queue for
  no_set: number; // Items none of the favorite creators have a set for
}

// For every item without a saved set, adds the best set from the favorite creators to the Download Queue
export const DownloadFromFavoriteCreators = async (): Promise<APIResponse<FavoriteCreatorsResult>> => {
  try {
    const response = await apiClient.post<APIResponse<FavoriteCreatorsResult>>(`/download/favorite-creators`);
    if (response.data.status === "error") {
      throw new Error(response.data.error?.message || "Unknown error downloading from the favorite creators");
    }
    return response.data;
  } catch (error) {
    log(
      "ERROR",
      "API - Download",
      "Favorite Creators",
      `Downloading from the favorite creators failed: ${error instanceof Error ? error.message : "Unknown error"}`,
      error
    );
    return ReturnErrorMessage<FavoriteCreatorsResult>(error);
  }
};
