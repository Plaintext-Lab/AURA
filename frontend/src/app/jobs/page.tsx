"use client";

import { ReturnErrorMessage } from "@/services/api-error-return";
import { GetAppConfigStatus } from "@/services/config/status";
import { UpdateAppConfig } from "@/services/config/update";
import type { JobInfo } from "@/services/jobs/get";
import { GetAllJobs } from "@/services/jobs/get";
import { RunJob } from "@/services/jobs/run";
import cronstrue from "cronstrue";
import { toast } from "sonner";

import { useEffect, useState } from "react";

import Link from "next/link";

import { ErrorMessage } from "@/components/shared/error-message";
import Loader from "@/components/shared/loader";
import { PopoverHelp } from "@/components/shared/popover-help";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";

import { cn } from "@/lib/cn";

import type { APIResponse } from "@/types/api/api-response";
import type { AppConfig, AppConfigJobSetting, AppConfigJobs } from "@/types/config/config";

type JobEdits = Record<string, AppConfigJobSetting>;

const validateCron = (expr: string): string | null => {
  const trimmed = expr.trim();
  if (!trimmed) return "Cron expression is required when enabled.";
  try {
    cronstrue.toString(trimmed);
    return null;
  } catch {
    return "Invalid cron expression. Use a site like crontab.guru to help you create and test your cron expressions.";
  }
};

// The backend formats timestamps as "YYYY-MM-DD HH:MM:SS" with no timezone
// suffix. Normalizing to an ISO-ish "T" separator makes Date parsing reliable
// across browsers (notably Safari, which can otherwise reject the space form).
const parseServerTimestamp = (value: string): number => {
  if (!value) return NaN;
  return new Date(value.replace(" ", "T")).getTime();
};

// Renders a millisecond duration as an at-a-glance label: "45s", "5 min", "6
// hours", "2 days".
const formatDuration = (ms: number): string => {
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;

  const minutes = Math.floor(ms / (60 * 1000));
  if (minutes < 60) return `${minutes} min`;

  const hours = Math.floor(ms / (60 * 60 * 1000));
  if (hours < 24) return `${hours} ${hours === 1 ? "hour" : "hours"}`;

  const days = Math.floor(ms / (24 * 60 * 60 * 1000));
  return `${days} ${days === 1 ? "day" : "days"}`;
};

const formatNextRun = (targetMs: number, nowMs: number): string => {
  const diffMs = targetMs - nowMs;
  if (diffMs <= 0) return "due now";
  return `in ${formatDuration(diffMs)}`;
};

const formatPrevRun = (targetMs: number, nowMs: number): string => {
  const diffMs = nowMs - targetMs;
  if (diffMs < 60 * 1000) return "just now";
  return `${formatDuration(diffMs)} ago`;
};

export default function JobsPage() {
  // States - Loading & Error
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<APIResponse<JobInfo[]> | null>(null);
  const [savingJobId, setSavingJobId] = useState<string | null>(null);

  // States - Data
  const [jobs, setJobs] = useState<JobInfo[]>([]);
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [edits, setEdits] = useState<JobEdits>({});

  // Ticks once a second so "Next Run" labels (and the under-a-minute countdown)
  // stay live without needing to refetch from the server.
  const [now, setNow] = useState<number>(() => Date.now());
  useEffect(() => {
    const interval = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(interval);
  }, []);

  const fetchJobs = async () => {
    try {
      setLoading(true);

      const [jobsResponse, configResponse] = await Promise.all([GetAllJobs(), GetAppConfigStatus()]);

      if (jobsResponse.status === "error") {
        setError(ReturnErrorMessage<JobInfo[]>(jobsResponse.error?.message || "Failed to fetch jobs"));
        setJobs([]);
        return;
      }
      if (configResponse.status === "error") {
        setError(ReturnErrorMessage<JobInfo[]>(configResponse.error?.message || "Failed to fetch configuration"));
        return;
      }

      const items = jobsResponse.data?.jobs || [];

      // Sort jobs by next_run date ascending
      items.sort((a, b) => {
        const dateA = parseServerTimestamp(a.next_run) || Infinity;
        const dateB = parseServerTimestamp(b.next_run) || Infinity;
        return dateA - dateB;
      });

      const newConfig = configResponse.data?.status.current_setup ?? null;

      setJobs(items);
      setConfig(newConfig);
      setEdits(Object.fromEntries(items.map((job) => [job.id, { enabled: job.enabled, cron: job.spec }])) as JobEdits);
      setError(null);
    } catch (e) {
      setError(ReturnErrorMessage<JobInfo[]>(e));
      setJobs([]);
    } finally {
      setLoading(false);
    }
  };

  const handleRunJobNowClick = async (job: JobInfo) => {
    const response = await RunJob(job.id);
    if (response.status === "error") {
      setError(ReturnErrorMessage<JobInfo[]>(response.error?.message || "Failed to trigger job"));
    } else {
      toast.success(`Job "${job.job_name}" triggered successfully`);
      void fetchJobs();
    }
  };

  const handleEditChange = <K extends keyof AppConfigJobSetting>(
    jobId: string,
    field: K,
    value: AppConfigJobSetting[K]
  ) => {
    setEdits((prev) => ({
      ...prev,
      [jobId]: { ...prev[jobId], [field]: value },
    }));
  };

  const handleSaveJobClick = async (job: JobInfo) => {
    if (!config) return;
    const edited = edits[job.id];
    if (!edited) return;

    if (edited.enabled) {
      const cronErr = validateCron(edited.cron);
      if (cronErr) {
        toast.error(cronErr);
        return;
      }
    }

    setSavingJobId(job.id);
    try {
      const newConfig: AppConfig = {
        ...config,
        jobs: {
          ...config.jobs,
          [job.id as keyof AppConfigJobs]: edited,
        },
      };

      const response = await UpdateAppConfig(newConfig);
      if (response.status === "error") {
        toast.error(response.error?.message || `Failed to update "${job.job_name}"`);
      } else if (response.status === "warn") {
        toast.warning("No changes detected.");
      } else {
        toast.success(`"${job.job_name}" updated successfully`);
        await fetchJobs();
      }
    } finally {
      setSavingJobId(null);
    }
  };

  useEffect(() => {
    void fetchJobs();
  }, []);

  const isJobDirty = (job: JobInfo) => {
    const edited = edits[job.id];
    if (!edited) return false;
    return edited.enabled !== job.enabled || edited.cron !== job.spec;
  };

  return (
    <div className="container mx-auto px-2 py-4 min-h-screen flex flex-col items-center">
      <Card className="w-full">
        <CardHeader>
          <div className="flex flex-col md:flex-row md:items-center md:justify-between w-full space-y-2 md:space-y-0">
            <h2 className="text-lg font-medium text-center md:text-left whitespace-nowrap">Scheduled Jobs</h2>

            <div className="flex flex-row flex-wrap justify-center md:justify-end gap-2 px-2 w-full max-w-full">
              <Button variant="outline" onClick={fetchJobs} disabled={loading}>
                Refresh
              </Button>
            </div>
          </div>
        </CardHeader>

        <CardContent>
          {loading ? (
            <Loader />
          ) : error ? (
            <ErrorMessage error={error} />
          ) : (!jobs || jobs.length === 0) && !error && !loading ? (
            <div className="w-full">
              <ErrorMessage error={ReturnErrorMessage<string>("No jobs found")} />
            </div>
          ) : (
            <div className="space-y-2">
              {jobs.map((job) => {
                const edited = edits[job.id] ?? { enabled: job.enabled, cron: job.spec };
                const dirty = isJobDirty(job);
                const cronErr = edited.enabled ? validateCron(edited.cron) : null;
                const cronTitle = !cronErr && edited.cron.trim() ? cronstrue.toString(edited.cron) : undefined;

                const nextRunLabel = job.enabled
                  ? job.next_run
                    ? formatNextRun(parseServerTimestamp(job.next_run), now)
                    : "—"
                  : "Disabled";
                const prevRunLabel = job.prev_run ? formatPrevRun(parseServerTimestamp(job.prev_run), now) : null;

                return (
                  <div key={job.id} className="rounded-lg border border-muted p-3 flex flex-col gap-2">
                    {/* Row 1: enabled toggle, name, description tooltip, next/prev run, run now */}
                    <div className="flex items-center gap-2 flex-wrap">
                      <Switch
                        checked={edited.enabled}
                        onCheckedChange={(v) => handleEditChange(job.id, "enabled", v)}
                        aria-label={`Toggle ${job.job_name}`}
                      />

                      <span className="text-sm font-medium truncate">{job.job_name || "Unknown Job"}</span>

                      {job.description && (
                        <PopoverHelp ariaLabel={`help-job-description-${job.id}`}>
                          <p>{job.description}</p>
                        </PopoverHelp>
                      )}

                      <div className="ml-auto flex items-center gap-3">
                        <span
                          className="text-xs text-muted-foreground font-mono whitespace-nowrap"
                          title={[job.next_run, job.prev_run].filter(Boolean).join(" · ") || undefined}
                        >
                          {nextRunLabel}
                          {prevRunLabel && <span className="hidden sm:inline"> · last {prevRunLabel}</span>}
                        </span>

                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => handleRunJobNowClick(job)}
                          disabled={loading}
                          className="shrink-0"
                        >
                          Run Now
                        </Button>
                      </div>
                    </div>

                    {/* Row 2: cron schedule + save */}
                    <div className="flex items-center gap-2 flex-wrap">
                      <Input
                        disabled={!edited.enabled}
                        placeholder="e.g. 0 3 * * *"
                        value={edited.cron}
                        onChange={(e) => handleEditChange(job.id, "cron", e.target.value)}
                        title={cronTitle}
                        className={cn("h-8 flex-1 min-w-[140px] font-mono text-sm", cronErr && "border-red-500")}
                      />

                      <PopoverHelp ariaLabel={`help-job-cron-${job.id}`}>
                        <p>
                          Cron expression format. Use the standard cron syntax to specify when the job should run. You
                          can use a site like{" "}
                          <Link
                            className="text-primary hover:underline"
                            href="https://crontab.guru/"
                            target="_blank"
                            rel="noopener noreferrer"
                          >
                            crontab.guru
                          </Link>{" "}
                          to help you create and test your cron expressions.
                        </p>
                      </PopoverHelp>

                      {dirty && (
                        <Button
                          size="sm"
                          onClick={() => handleSaveJobClick(job)}
                          disabled={savingJobId === job.id || !!cronErr}
                          className="shrink-0"
                        >
                          {savingJobId === job.id ? "Saving..." : "Save"}
                        </Button>
                      )}
                    </div>

                    {cronErr && <p className="text-xs text-red-500">{cronErr}</p>}
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
