"use client";

import { ReturnErrorMessage } from "@/services/api-error-return";
import { DeleteTempImages } from "@/services/images/api-images-actions";
import { toast } from "sonner";

import React, { useEffect, useRef } from "react";

import { ConfirmDestructiveDialogActionButton } from "@/components/shared/dialog-destructive-action";
import { PopoverHelp } from "@/components/shared/popover-help";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectScrollDownButton,
  SelectScrollUpButton,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

import { cn } from "@/lib/cn";

import type { AppConfigImages } from "@/types/config/config";

const EPISODE_NAMING_CONVENTION_OPTIONS = ["match", "static"];

interface ConfigSectionImagesProps {
  value: AppConfigImages;
  editing: boolean;
  showActions?: boolean; // Show actions hidden while editing, as Settings is always editable
  dirtyFields?: {
    cache_images?: { enabled?: boolean };
    save_images_locally?: {
      enabled?: boolean;
      path?: boolean;
      episode_naming_convention?: boolean;
      running_on_windows?: boolean;
    };
  };
  onChange: <K extends keyof AppConfigImages, F extends keyof AppConfigImages[K]>(
    group: K,
    field: F,
    value: AppConfigImages[K][F]
  ) => void;
  errorsUpdate?: (errors: Partial<Record<keyof AppConfigImages, string>>) => void;
  mediaServerType?: string;
}

export const ConfigSectionImages: React.FC<ConfigSectionImagesProps> = ({
  value,
  editing,
  showActions = false,
  dirtyFields = {},
  onChange,
  errorsUpdate,
  mediaServerType,
}) => {
  const prevErrorsRef = useRef<string>("{}");

  const clearTempImagesFolder = async () => {
    try {
      const response = await DeleteTempImages();
      if (response.status === "error") {
        toast.error(response.error?.message || "Failed to clear temp images folder");
        return;
      }
      toast.success(response.data?.message || "Temp images folder cleared successfully");
    } catch (error) {
      const errorResponse = ReturnErrorMessage<void>(error);
      toast.error(errorResponse.error?.message || "An unexpected error occurred");
    }
  };

  const errors = React.useMemo<Partial<Record<keyof AppConfigImages, string>>>(() => {
    const errs: Partial<Record<keyof AppConfigImages, string>> = {};

    // Validate SaveImagesLocally.EpisodeNamingConvention
    if (mediaServerType && value.save_images_locally.enabled) {
      if (!value.save_images_locally.episode_naming_convention) {
        errs.save_images_locally = "Episode naming convention is required.";
      } else {
        if (!EPISODE_NAMING_CONVENTION_OPTIONS.includes(value.save_images_locally.episode_naming_convention)) {
          errs.save_images_locally = `Episode naming convention must be one of: ${EPISODE_NAMING_CONVENTION_OPTIONS.join(", ")}.`;
        }
      }
    }

    return errs;
  }, [mediaServerType, value.save_images_locally.enabled, value.save_images_locally.episode_naming_convention]);

  // Emit errors upward
  useEffect(() => {
    if (!errorsUpdate) return;
    const serialized = JSON.stringify(errors);
    if (serialized === prevErrorsRef.current) return;
    prevErrorsRef.current = serialized;
    errorsUpdate(errors);
  }, [errors, errorsUpdate]);

  return (
    <Card className="p-5">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-blue-500">Images</h2>
        <ConfirmDestructiveDialogActionButton
          hidden={editing && !showActions}
          onConfirm={clearTempImagesFolder}
          title="Clear Temp Images Folder?"
          description="This will permanently delete all temporary images. Are you sure you want to continue?"
          confirmText="Yes, Clear Images"
          cancelText="Cancel"
          className="text-destructive border-1 shadow-none hover:text-red-500 cursor-pointer"
          variant="ghost"
        >
          Clear Temp Images
        </ConfirmDestructiveDialogActionButton>
      </div>
      {/* Cache Images */}
      <div
        className={cn(
          "flex items-center justify-between border rounded-md p-3 transition",
          "border-muted",
          dirtyFields.cache_images?.enabled && "border-amber-500"
        )}
      >
        <Label className="mr-2">Cache Images</Label>
        <div className="flex items-center gap-2">
          <Switch
            disabled={!editing}
            checked={value.cache_images.enabled}
            onCheckedChange={(v) => onChange("cache_images", "enabled", v)}
          />
          {editing && (
            <PopoverHelp ariaLabel="help-images-cache">
              <p>Store downloaded artwork locally to reduce external requests and speed repeat access.</p>
            </PopoverHelp>
          )}
        </div>
      </div>

      {/* Save Images Locally */}
      <div
        className={cn(
          "border rounded-md p-3 transition",
          "border-muted",
          dirtyFields.save_images_locally?.enabled && "border-amber-500"
        )}
      >
        <div className="flex items-center justify-between mb-2">
          <Label className="mr-2">Save Images Locally</Label>
          <div className="flex items-center gap-2">
            <Switch
              disabled={!editing}
              checked={!!value.save_images_locally.enabled}
              onCheckedChange={(v) => onChange("save_images_locally", "enabled", v)}
            />
            {editing && (
              <PopoverHelp ariaLabel="help-images-save-next-to-content">
                <p>
                  Save images to a local folder on the server. This is useful for not relying on your Media Server
                  database. Make sure the path is accessible by the Aura server.
                </p>
              </PopoverHelp>
            )}
          </div>
        </div>

        {value.save_images_locally.enabled && (
          <div
            className={cn("mt-2", dirtyFields.save_images_locally?.enabled && "border border-amber-500 rounded-md p-2")}
          >
            <div className="flex items-center justify-between mb-2">
              <Label className="mr-2">Path</Label>
              {editing && (
                <PopoverHelp ariaLabel="help-images-save-path">
                  <p>
                    Enter the local folder path where images should be saved. This must be accessible by the Aura
                    server. Leave this blank if you want to save images next to the content.
                  </p>
                </PopoverHelp>
              )}
            </div>
            <Input
              type="text"
              disabled={!editing}
              value={value.save_images_locally.path || ""}
              onChange={(e) => onChange("save_images_locally", "path", e.target.value)}
              className={cn(
                "w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50 transition",
                dirtyFields.save_images_locally?.path && "border-amber-500"
              )}
              placeholder="/path/to/images"
            />
          </div>
        )}

        {value.save_images_locally.enabled && (
          <div className={cn("space-y-1 mt-4")}>
            <div className="flex items-center justify-between">
              <Label>Episode Naming Convention</Label>
              {editing && (
                <PopoverHelp ariaLabel="help-media-server-episode-naming-convention">
                  <div className="space-y-3">
                    <div>
                      <p className="font-medium mb-1">Episode Naming Convention</p>
                      <p className="text-muted-foreground">How episode files are named.</p>
                    </div>
                    <ul className="space-y-1">
                      <li className="flex items-center gap-2">
                        <span className="inline-flex h-5 items-center rounded-sm bg-muted px-2 font-mono ">match</span>
                        <span>Some Episode Title S01E01.jpg</span>
                      </li>
                      <li className="flex items-center gap-2">
                        <span className="inline-flex h-5 items-center rounded-sm bg-muted px-2 font-mono">static</span>
                        <span>S01E01.jpg</span>
                      </li>
                    </ul>
                    <p className="text-muted-foreground">Used for file naming logic.</p>
                  </div>
                </PopoverHelp>
              )}
            </div>
            <Select
              disabled={!editing}
              value={value.save_images_locally.episode_naming_convention || ""}
              onValueChange={(v) => onChange("save_images_locally", "episode_naming_convention", v)}
            >
              <SelectTrigger
                id="media-server-episode-naming-convention-trigger"
                className={cn(
                  "w-full",
                  dirtyFields.save_images_locally?.episode_naming_convention && "border-amber-500"
                )}
              >
                <SelectValue placeholder="Select convention..." />
              </SelectTrigger>
              <SelectContent>
                {EPISODE_NAMING_CONVENTION_OPTIONS.map((o) => (
                  <SelectItem key={o} value={o}>
                    {o}
                  </SelectItem>
                ))}
                <SelectScrollUpButton />
                <SelectScrollDownButton />
              </SelectContent>
            </Select>
          </div>
        )}

        {value.save_images_locally.enabled && (
          <div
            className={cn(
              "mt-4 flex items-center justify-between rounded-md border p-3",
              dirtyFields.save_images_locally?.running_on_windows && "border-amber-500"
            )}
          >
            <Label className="mr-2">Running on Windows</Label>
            <div className="flex items-center gap-2">
              <Switch
                disabled={!editing}
                checked={!!value.save_images_locally.running_on_windows}
                onCheckedChange={(v) => onChange("save_images_locally", "running_on_windows", v)}
              />
              {editing && (
                <PopoverHelp ariaLabel="help-images-running-on-windows">
                  <p className="mb-2">Turn on if your media server runs on Windows.</p>
                  <p className="text-muted-foreground">
                    Windows paths from the media server (e.g. <span className="font-mono">C:\Movies</span>) are
                    converted to paths aura can use (e.g. <span className="font-mono">/C/Movies</span>). Make sure those
                    paths are mounted in the aura container.
                  </p>
                </PopoverHelp>
              )}
            </div>
          </div>
        )}
      </div>
    </Card>
  );
};
