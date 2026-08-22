"use client";

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { ValidateActivitySourceInfo } from "@/services/validation/activity";
import { TriggerActivitySync } from "@/services/activity";

import { PopoverHelp } from "@/components/shared/popover-help";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

import { cn } from "@/lib/cn";

import type { AppConfigActivitySource } from "@/types/config/config";

interface ConfigSectionActivityProps {
  value: AppConfigActivitySource;
  editing: boolean;
  onChange: <K extends keyof AppConfigActivitySource>(field: K, value: AppConfigActivitySource[K]) => void;
  errorsUpdate?: (errors: Record<string, string>) => void;
  configAlreadyLoaded: boolean;
}

const PROVIDERS = ["tautulli", "tracearr"] as const;

type ConnectionStatus = { status: "ok" | "error" | "unknown" };

const STATUS_BG: Record<string, string> = {
  ok: "bg-green-500",
  error: "bg-red-500",
  unknown: "bg-gray-400",
};

export const ConfigSectionActivity: React.FC<ConfigSectionActivityProps> = ({
  value,
  editing,
  onChange,
  errorsUpdate,
  configAlreadyLoaded,
}) => {
  const prevErrorsRef = useRef<string>("");
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>({ status: "unknown" });
  const [testing, setTesting] = useState(false);
  const [syncing, setSyncing] = useState(false);

  const errors = useMemo(() => {
    const errs: Record<string, string> = {};
    if (!value.enabled) return errs;

    if (!value.provider) {
      errs["Provider"] = "Provider is required";
    } else if (!PROVIDERS.includes(value.provider as (typeof PROVIDERS)[number])) {
      errs["Provider"] = "Provider must be tautulli or tracearr";
    }

    if (!value.base_url) {
      errs["BaseURL"] = "Base URL is required";
    } else {
      try {
        new URL(value.base_url);
      } catch {
        errs["BaseURL"] = "Base URL must be a valid URL";
      }
    }

    if (!value.api_token) {
      errs["ApiToken"] = "API token is required";
    }

    if (value.activity_window_days <= 0) {
      errs["ActivityWindowDays"] = "Activity window must be a positive number";
    }

    return errs;
  }, [value]);

  useEffect(() => {
    const errStr = JSON.stringify(errors);
    if (errStr !== prevErrorsRef.current) {
      prevErrorsRef.current = errStr;
      errorsUpdate?.(errors);
    }
  }, [errors, errorsUpdate]);

  const handleTestConnection = useCallback(async () => {
    if (!value.provider || !value.base_url || !value.api_token) return;
    setTesting(true);
    setConnectionStatus({ status: "unknown" });
    const result = await ValidateActivitySourceInfo(value, true);
    setConnectionStatus({ status: result.valid ? "ok" : "error" });
    setTesting(false);
  }, [value]);

  const handleSyncNow = useCallback(async () => {
    setSyncing(true);
    await TriggerActivitySync(true);
    setSyncing(false);
  }, []);

  const field = <K extends keyof AppConfigActivitySource>(key: K, val: AppConfigActivitySource[K]) => {
    onChange(key, val);
  };

  return (
    <Card className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold">Activity Source</h3>
          <p className="text-sm text-muted-foreground">
            Connect Tautulli or Tracearr to provide media activity data. No user, device or IP information is stored.
          </p>
        </div>
        <Switch
          checked={value.enabled}
          onCheckedChange={(checked) => field("enabled", checked)}
          disabled={!editing}
        />
      </div>

      {value.enabled && (
        <div className="space-y-4">
          {/* Provider */}
          <div className="space-y-2">
            <Label>
              Provider
              <PopoverHelp content="Choose the activity provider: Tautulli (Plex statistics) or Tracearr." />
            </Label>
            <Select
              value={value.provider || ""}
              onValueChange={(v) => field("provider", v)}
              disabled={!editing}
            >
              <SelectTrigger className={cn(errors["Provider"] && "border-red-500")}>
                <SelectValue placeholder="Select provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="tautulli">Tautulli</SelectItem>
                <SelectItem value="tracearr">Tracearr</SelectItem>
              </SelectContent>
            </Select>
            {errors["Provider"] && <p className="text-xs text-red-500">{errors["Provider"]}</p>}
          </div>

          {/* Base URL */}
          <div className="space-y-2">
            <Label>
              Base URL
              <PopoverHelp content="The base URL of your provider, e.g. http://tautulli:8181" />
            </Label>
            <Input
              value={value.base_url}
              onChange={(e) => field("base_url", e.target.value)}
              disabled={!editing}
              placeholder="http://tautulli:8181"
              className={cn(errors["BaseURL"] && "border-red-500")}
            />
            {errors["BaseURL"] && <p className="text-xs text-red-500">{errors["BaseURL"]}</p>}
          </div>

          {/* API Token */}
          <div className="space-y-2">
            <Label>
              API Token
              <PopoverHelp content="Your provider API key or token. It is stored securely and masked in responses." />
            </Label>
            <Input
              type="password"
              value={value.api_token}
              onChange={(e) => field("api_token", e.target.value)}
              disabled={!editing}
              placeholder={configAlreadyLoaded ? "••••••••" : "Enter API token"}
              className={cn(errors["ApiToken"] && "border-red-500")}
            />
            {errors["ApiToken"] && <p className="text-xs text-red-500">{errors["ApiToken"]}</p>}
          </div>

          {/* Tracearr Server ID */}
          {value.provider === "tracearr" && (
            <div className="space-y-2">
              <Label>
                Tracearr Server ID
                <PopoverHelp content="The UUID of the Tracearr server to query. Required when Tracearr monitors multiple servers." />
              </Label>
              <Input
                value={value.tracearr_server_id ?? ""}
                onChange={(e) => field("tracearr_server_id", e.target.value)}
                disabled={!editing}
                placeholder="Server UUID (optional if only one server)"
              />
            </div>
          )}

          {/* Activity Window */}
          <div className="space-y-2">
            <Label>
              Activity Window (days)
              <PopoverHelp content="Number of days of history to consider for play counts. Default: 30." />
            </Label>
            <Input
              type="number"
              min={1}
              value={value.activity_window_days}
              onChange={(e) => field("activity_window_days", parseInt(e.target.value, 10) || 30)}
              disabled={!editing}
              className={cn(errors["ActivityWindowDays"] && "border-red-500")}
            />
            {errors["ActivityWindowDays"] && (
              <p className="text-xs text-red-500">{errors["ActivityWindowDays"]}</p>
            )}
          </div>

          {/* Refresh Interval */}
          <div className="space-y-2">
            <Label>
              Refresh Interval (cron)
              <PopoverHelp content="Cron expression for how often to sync activity data. Default: every 30 minutes." />
            </Label>
            <Input
              value={value.refresh_interval}
              onChange={(e) => field("refresh_interval", e.target.value)}
              disabled={!editing}
              placeholder="*/30 * * * *"
            />
          </div>

          {/* Action buttons */}
          <div className="flex gap-2 flex-wrap">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleTestConnection}
              disabled={testing || !value.provider || !value.base_url || !value.api_token}
              className="flex items-center gap-2"
            >
              <span
                className={cn(
                  "inline-block w-2 h-2 rounded-full",
                  STATUS_BG[connectionStatus.status]
                )}
              />
              {testing ? "Testing..." : "Test Connection"}
            </Button>

            {configAlreadyLoaded && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleSyncNow}
                disabled={syncing}
              >
                {syncing ? "Syncing..." : "Sync Now"}
              </Button>
            )}
          </div>
        </div>
      )}
    </Card>
  );
};

export default ConfigSectionActivity;
