import apiClient from "@/services/api-client";
import { ReturnErrorMessage } from "@/services/api-error-return";
import { toast } from "sonner";
import { log } from "@/lib/logger";
import type { APIResponse } from "@/types/api/api-response";
import type { ActivitySummariesResponse } from "@/types/activity/activity";

export const GetActivitySummaries = async (): Promise<ActivitySummariesResponse | null> => {
  log("INFO", "API - Activity", "Summaries", "Fetching activity summaries");
  try {
    const response = await apiClient.get<APIResponse<ActivitySummariesResponse>>(`/activity/summaries`);
    if (response.data.status === "error") {
      log("ERROR", "API - Activity", "Summaries", "Error fetching summaries", response.data.error);
      return null;
    }
    return response.data.data ?? null;
  } catch (error) {
    const err = ReturnErrorMessage<ActivitySummariesResponse>(error);
    log("ERROR", "API - Activity", "Summaries", `Request failed: ${err.error?.message}`, error);
    return null;
  }
};

export const TriggerActivitySync = async (showToast = true): Promise<boolean> => {
  log("INFO", "API - Activity", "Sync", "Triggering manual activity sync");
  let loadingToast: string | number | undefined;
  try {
    if (showToast) loadingToast = toast.loading("Starting activity sync...");
    const response = await apiClient.post<APIResponse<{ message: string }>>(`/activity/sync`, {});
    if (showToast && loadingToast) toast.dismiss(loadingToast);

    if (response.data.status === "error") {
      const msg = response.data.error?.message || "Failed to trigger activity sync.";
      if (showToast) toast.error(msg, { duration: 2000 });
      return false;
    }
    if (showToast) toast.success("Activity sync started", { duration: 1500 });
    return true;
  } catch (error) {
    if (showToast && loadingToast) toast.dismiss(loadingToast);
    const err = ReturnErrorMessage<{ message: string }>(error);
    const msg = err.error?.message || "Failed to trigger activity sync.";
    if (showToast) toast.error(msg, { duration: 2000 });
    log("ERROR", "API - Activity", "Sync", `Request failed: ${msg}`, error);
    return false;
  }
};
