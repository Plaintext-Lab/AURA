"use client";

import { ReturnErrorMessage } from "@/services/api-error-return";
import { UpdateAppConfigSection } from "@/services/config/update-section";
import type { JobInfo } from "@/services/jobs/get";
import { GetAllJobs } from "@/services/jobs/get";
import { RunJob } from "@/services/jobs/run";
import cronstrue from "cronstrue";
import { CalendarClock, History, Loader2, Play, RefreshCcw } from "lucide-react";
import { toast } from "sonner";

import { useEffect, useState } from "react";

import Link from "next/link";

import { ErrorMessage } from "@/components/shared/error-message";
import Loader from "@/components/shared/loader";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";

import { cn } from "@/lib/cn";

import type { APIResponse } from "@/types/api/api-response";
import type { AppConfigJobSetting } from "@/types/config/config";

type JobEdits = Record<string, AppConfigJobSetting>;

const validateCron = (expr: string): string | null => {
  const trimmed = expr.trim();
  if (!trimmed) return "Cron expression is required when enabled.";
  try {
    cronstrue.toString(trimmed);
    return null;
  } catch {
    return "Not a valid cron expression.";
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
  const [runningJobId, setRunningJobId] = useState<string | null>(null);

  // States - Data
  const [jobs, setJobs] = useState<JobInfo[]>([]);
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

      const jobsResponse = await GetAllJobs();
      if (jobsResponse.status === "error") {
        setError(ReturnErrorMessage<JobInfo[]>(jobsResponse.error?.message || "Failed to fetch jobs"));
        setJobs([]);
        return;
      }

      const items = jobsResponse.data?.jobs || [];

      // Sort jobs by next_run date ascending
      items.sort((a, b) => {
        const dateA = parseServerTimestamp(a.next_run) || Infinity;
        const dateB = parseServerTimestamp(b.next_run) || Infinity;
        return dateA - dateB;
      });

      setJobs(items);
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
    setRunningJobId(job.id);
    const response = await RunJob(job.id);
    setRunningJobId(null);
    if (response.status === "error") {
      toast.error(response.error?.message || `Failed to run "${job.job_name}"`);
    } else {
      toast.success(`"${job.job_name}" started`);
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

  const handleDiscardClick = (job: JobInfo) => {
    setEdits((prev) => ({ ...prev, [job.id]: { enabled: job.enabled, cron: job.spec } }));
  };

  const handleSaveJobClick = async (job: JobInfo) => {
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
      const response = await UpdateAppConfigSection("jobs", { [job.id]: { ...edited, cron: edited.cron.trim() } });
      if (response.status === "error") {
        toast.error(response.error?.message || `Failed to save "${job.job_name}"`);
      } else if (!response.data?.changed) {
        toast.warning("No changes to save");
      } else {
        toast.success(`"${job.job_name}" saved`);
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
    <div className="container mx-auto max-w-6xl p-4 sm:p-6 lg:p-8">
      <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1">
          <h1 className="text-3xl font-bold">Jobs</h1>
          <p className="text-sm text-muted-foreground">
            Background jobs that keep aura up to date. Each job runs on its own schedule, written as a cron expression.
            You can use{" "}
            <Link
              className="text-primary hover:underline"
              href="https://crontab.guru/"
              target="_blank"
              rel="noopener noreferrer"
            >
              crontab.guru
            </Link>{" "}
            to create and test them.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={fetchJobs}
          disabled={loading}
          className="shrink-0 self-start cursor-pointer"
        >
          <RefreshCcw className={cn("h-4 w-4", loading && "animate-spin")} />
          Refresh
        </Button>
      </div>

      {loading && jobs.length === 0 ? (
        <Loader />
      ) : error ? (
        <ErrorMessage error={error} />
      ) : jobs.length === 0 ? (
        <ErrorMessage error={ReturnErrorMessage<string>("No jobs found")} />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {jobs.map((job) => {
            const edited = edits[job.id] ?? { enabled: job.enabled, cron: job.spec };
            const dirty = isJobDirty(job);
            const cronErr = edited.enabled ? validateCron(edited.cron) : null;
            const schedule = !cronErr && edited.cron.trim() ? cronstrue.toString(edited.cron.trim()) : null;

            const nextRunLabel = job.enabled
              ? job.next_run
                ? formatNextRun(parseServerTimestamp(job.next_run), now)
                : "—"
              : "Off";
            const prevRunLabel = job.prev_run ? formatPrevRun(parseServerTimestamp(job.prev_run), now) : null;

            return (
              <Card key={job.id} className={cn("gap-4 p-4", dirty && "border-amber-500")}>
                {/* Name, description and on/off */}
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 space-y-1">
                    <div className="flex items-center gap-2">
                      <h2 className="font-semibold leading-tight">{job.job_name || "Unknown Job"}</h2>
                      {!job.enabled && <Badge variant="secondary">Off</Badge>}
                    </div>
                    {job.description && <p className="text-sm text-muted-foreground">{job.description}</p>}
                  </div>
                  <Switch
                    checked={edited.enabled}
                    onCheckedChange={(v) => handleEditChange(job.id, "enabled", v)}
                    aria-label={`Turn ${job.job_name} on or off`}
                    className="mt-0.5 cursor-pointer"
                  />
                </div>

                {/* Schedule */}
                <div className="space-y-1">
                  <Label htmlFor={`job-cron-${job.id}`} className="text-xs text-muted-foreground">
                    Schedule
                  </Label>
                  <Input
                    id={`job-cron-${job.id}`}
                    disabled={!edited.enabled}
                    placeholder="0 3 * * *"
                    value={edited.cron}
                    onChange={(e) => handleEditChange(job.id, "cron", e.target.value)}
                    className={cn("font-mono", cronErr && "border-red-500", dirty && !cronErr && "border-amber-500")}
                  />
                  {cronErr ? (
                    <p className="text-xs text-red-500">{cronErr}</p>
                  ) : (
                    schedule && edited.enabled && <p className="text-xs text-muted-foreground">{schedule}</p>
                  )}
                </div>

                {/* Next and last run, and actions */}
                <div className="mt-auto flex flex-wrap items-center justify-between gap-3 border-t pt-3">
                  <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1" title={job.next_run || undefined}>
                      <CalendarClock className="h-3.5 w-3.5" />
                      Next: {nextRunLabel}
                    </span>
                    {prevRunLabel && (
                      <span className="flex items-center gap-1" title={job.prev_run || undefined}>
                        <History className="h-3.5 w-3.5" />
                        Last: {prevRunLabel}
                      </span>
                    )}
                  </div>

                  <div className="ml-auto flex items-center gap-2">
                    {dirty ? (
                      <>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDiscardClick(job)}
                          disabled={savingJobId === job.id}
                          className="cursor-pointer"
                        >
                          Discard
                        </Button>
                        <Button
                          size="sm"
                          onClick={() => handleSaveJobClick(job)}
                          disabled={savingJobId === job.id || !!cronErr}
                          className="cursor-pointer"
                        >
                          {savingJobId === job.id && <Loader2 className="h-4 w-4 animate-spin" />}
                          {savingJobId === job.id ? "Saving..." : "Save"}
                        </Button>
                      </>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleRunJobNowClick(job)}
                        disabled={runningJobId === job.id}
                        className="cursor-pointer"
                      >
                        {runningJobId === job.id ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <Play className="h-4 w-4" />
                        )}
                        Run Now
                      </Button>
                    )}
                  </div>
                </div>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
