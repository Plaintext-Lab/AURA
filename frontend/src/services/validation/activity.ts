import apiClient from "@/services/api-client";
import { ReturnErrorMessage } from "@/services/api-error-return";
import { toast } from "sonner";

import { log } from "@/lib/logger";

import type { APIResponse } from "@/types/api/api-response";
import type { AppConfigActivitySource } from "@/types/config/config";

export interface ValidateActivitySource_Request {
  activity_source: AppConfigActivitySource;
}

export interface ValidateActivitySource_Response {
  valid: boolean;
  message: string;
  provider?: {
    name: string;
    version?: string;
    server_id?: string;
  };
}

export const ValidateActivitySourceInfo = async (
  activitySource: AppConfigActivitySource,
  showToast = true
): Promise<ValidateActivitySource_Response> => {
  log("INFO", "API - Settings", "Activity", "Validating activity source connection info");

  let loadingToast: string | number | undefined;
  try {
    if (showToast) {
      loadingToast = toast.loading(`Checking connection to ${activitySource.provider || "activity provider"}...`);
    }

    const req: ValidateActivitySource_Request = { activity_source: activitySource };
    const response = await apiClient.post<APIResponse<ValidateActivitySource_Response>>(
      `/validate/activity`,
      req
    );

    if (showToast && loadingToast) toast.dismiss(loadingToast);

    if (response.data.status === "error") {
      const msg = response.data.error?.message || "Couldn't connect to activity provider. Check the URL and token.";
      if (showToast) toast.error(msg, { duration: 2000 });
      return { valid: false, message: msg };
    }

    const data = response.data.data;
    if (!data) {
      const msg = "Couldn't connect to activity provider. Check the URL and token.";
      if (showToast) toast.error(msg, { duration: 2000 });
      return { valid: false, message: msg };
    }

    if (showToast) {
      if (data.valid)
        toast.success(data.message || `Successfully connected to ${activitySource.provider}`, { duration: 1500 });
      else
        toast.error(data.message || "Couldn't connect to activity provider.", { duration: 2000 });
    }

    log("INFO", "API - Settings", "Activity", "Validation response received", data);
    return data;
  } catch (error) {
    if (showToast) {
      if (loadingToast) toast.dismiss(loadingToast);
    }

    const errorResponse = ReturnErrorMessage<ValidateActivitySource_Response>(error);
    const msg = errorResponse.error?.message || "Couldn't connect to activity provider.";

    log("ERROR", "API - Settings", "Activity", `Validation request failed: ${error instanceof Error ? error.message : "Unknown error"}`, error);

    if (showToast) toast.error(msg, { duration: 2000 });
    return { valid: false, message: msg };
  }
};
