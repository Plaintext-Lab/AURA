import apiClient from "@/services/api-client";
import { ReturnErrorMessage } from "@/services/api-error-return";

import { log } from "@/lib/logger";

import type { APIResponse } from "@/types/api/api-response";
import type {
  LibraryGroup,
  LibraryGroupPreview,
  ReconcileDecision,
} from "@/types/database/library-group";

// ─────────────────────────────────────────────────────────────────────────────
// Get all library groups
// ─────────────────────────────────────────────────────────────────────────────

export interface GetLibraryGroups_Response {
  groups: LibraryGroup[];
}

export const GetLibraryGroups = async (): Promise<APIResponse<GetLibraryGroups_Response>> => {
  log("INFO", "API - Library Groups", "Get", "Fetching all library groups");
  try {
    const response = await apiClient.get<APIResponse<GetLibraryGroups_Response>>(
      `/db/library-groups`
    );
    if (response.data.status === "error") {
      throw new Error(response.data.error?.message || "Unknown error");
    }
    return response.data;
  } catch (error) {
    log("ERROR", "API - Library Groups", "Get", "Failed to fetch library groups", error);
    return ReturnErrorMessage<GetLibraryGroups_Response>(error);
  }
};

// ─────────────────────────────────────────────────────────────────────────────
// Create a library group
// ─────────────────────────────────────────────────────────────────────────────

export interface UpsertLibraryGroup_Request {
  name: string;
  media_type: "movie" | "show";
  library_ids: string[];
}

export interface CreateLibraryGroup_Response {
  group: LibraryGroup;
}

export const CreateLibraryGroup = async (
  req: UpsertLibraryGroup_Request
): Promise<APIResponse<CreateLibraryGroup_Response>> => {
  log("INFO", "API - Library Groups", "Create", `Creating library group '${req.name}'`);
  try {
    const response = await apiClient.post<APIResponse<CreateLibraryGroup_Response>>(
      `/db/library-groups`,
      req
    );
    if (response.data.status === "error") {
      throw new Error(response.data.error?.message || "Unknown error");
    }
    return response.data;
  } catch (error) {
    log("ERROR", "API - Library Groups", "Create", "Failed to create library group", error);
    return ReturnErrorMessage<CreateLibraryGroup_Response>(error);
  }
};

// ─────────────────────────────────────────────────────────────────────────────
// Update a library group
// ─────────────────────────────────────────────────────────────────────────────

export interface UpdateLibraryGroup_Response {
  group: LibraryGroup;
}

export const UpdateLibraryGroup = async (
  id: string,
  req: UpsertLibraryGroup_Request
): Promise<APIResponse<UpdateLibraryGroup_Response>> => {
  log("INFO", "API - Library Groups", "Update", `Updating library group '${id}'`);
  try {
    const response = await apiClient.put<APIResponse<UpdateLibraryGroup_Response>>(
      `/db/library-groups/${id}`,
      req
    );
    if (response.data.status === "error") {
      throw new Error(response.data.error?.message || "Unknown error");
    }
    return response.data;
  } catch (error) {
    log("ERROR", "API - Library Groups", "Update", "Failed to update library group", error);
    return ReturnErrorMessage<UpdateLibraryGroup_Response>(error);
  }
};

// ─────────────────────────────────────────────────────────────────────────────
// Delete a library group
// ─────────────────────────────────────────────────────────────────────────────

export interface DeleteLibraryGroup_Response {
  result: string;
}

export const DeleteLibraryGroup = async (
  id: string
): Promise<APIResponse<DeleteLibraryGroup_Response>> => {
  log("INFO", "API - Library Groups", "Delete", `Deleting library group '${id}'`);
  try {
    const response = await apiClient.delete<APIResponse<DeleteLibraryGroup_Response>>(
      `/db/library-groups/${id}`
    );
    if (response.data.status === "error") {
      throw new Error(response.data.error?.message || "Unknown error");
    }
    return response.data;
  } catch (error) {
    log("ERROR", "API - Library Groups", "Delete", "Failed to delete library group", error);
    return ReturnErrorMessage<DeleteLibraryGroup_Response>(error);
  }
};

// ─────────────────────────────────────────────────────────────────────────────
// Preview a library group (conflicts / match count)
// ─────────────────────────────────────────────────────────────────────────────

export interface PreviewLibraryGroup_Response {
  preview: LibraryGroupPreview;
}

export const PreviewLibraryGroup = async (
  id: string,
  req: UpsertLibraryGroup_Request
): Promise<APIResponse<PreviewLibraryGroup_Response>> => {
  log("INFO", "API - Library Groups", "Preview", `Previewing library group '${id}'`);
  try {
    const response = await apiClient.post<APIResponse<PreviewLibraryGroup_Response>>(
      `/db/library-groups/${id}/preview`,
      req
    );
    if (response.data.status === "error") {
      throw new Error(response.data.error?.message || "Unknown error");
    }
    return response.data;
  } catch (error) {
    log("ERROR", "API - Library Groups", "Preview", "Failed to preview library group", error);
    return ReturnErrorMessage<PreviewLibraryGroup_Response>(error);
  }
};

// ─────────────────────────────────────────────────────────────────────────────
// Reconcile a library group
// ─────────────────────────────────────────────────────────────────────────────

export interface ReconcileLibraryGroup_Request {
  policy_decisions: Record<string, ReconcileDecision>;
}

export interface ReconcileLibraryGroup_Response {
  reconciled: number;
  skipped: number;
}

export const ReconcileLibraryGroup = async (
  id: string,
  req: ReconcileLibraryGroup_Request
): Promise<APIResponse<ReconcileLibraryGroup_Response>> => {
  log("INFO", "API - Library Groups", "Reconcile", `Reconciling library group '${id}'`);
  try {
    const response = await apiClient.post<APIResponse<ReconcileLibraryGroup_Response>>(
      `/db/library-groups/${id}/reconcile`,
      req
    );
    if (response.data.status === "error") {
      throw new Error(response.data.error?.message || "Unknown error");
    }
    return response.data;
  } catch (error) {
    log("ERROR", "API - Library Groups", "Reconcile", "Failed to reconcile library group", error);
    return ReturnErrorMessage<ReconcileLibraryGroup_Response>(error);
  }
};
