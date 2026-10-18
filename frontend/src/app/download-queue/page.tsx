"use client";

import { GetDownloadHistory } from "@/services/downloads/history-get";
import { RemoveAllDownloadHistory } from "@/services/downloads/history-remove";
import { GetAllDownloadQueueItems } from "@/services/downloads/queue-get";
import { RemoveAllFromQueue } from "@/services/downloads/queue-remove";
import { subscribeToDownloadQueue } from "@/services/downloads/queue-stream";
import { toast } from "sonner";

import React, { useCallback, useEffect, useState } from "react";

import Link from "next/link";

import { ConfirmDestructiveDialogActionButton } from "@/components/shared/dialog-destructive-action";
import DownloadEntryCard from "@/components/shared/download-entry";
import { DownloadQueueProgressPanel } from "@/components/shared/download-queue-progress";
import { ErrorMessage } from "@/components/shared/error-message";
import Loader from "@/components/shared/loader";
import { ResponsiveGrid } from "@/components/shared/responsive-grid";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { H2 } from "@/components/ui/typography";

import type { APIResponse } from "@/types/api/api-response";
import type { DownloadHistoryEntry, DownloadQueueJob, DownloadQueueProgress } from "@/types/database/download-queue";

const DownloadQueuePage: React.FC = () => {
  // States - Queue
  const [jobs, setJobs] = useState<DownloadQueueJob[]>([]);
  const [progress, setProgress] = useState<DownloadQueueProgress | null>(null);
  const [queueLoading, setQueueLoading] = useState(true);
  const [queueError, setQueueError] = useState<APIResponse<unknown> | null>(null);

  // States - History
  const [historyEntries, setHistoryEntries] = useState<DownloadHistoryEntry[]>([]);
  const [historyLoading, setHistoryLoading] = useState(true);
  const [historyError, setHistoryError] = useState<APIResponse<unknown> | null>(null);

  const fetchHistory = useCallback(async () => {
    try {
      setHistoryLoading(true);
      const response = await GetDownloadHistory();
      if (response.status === "error") {
        setHistoryError(response);
        return;
      }
      setHistoryEntries(response.data?.entries || []);
      setHistoryError(null);
    } finally {
      setHistoryLoading(false);
    }
  }, []);

  // Initial queue load (SSE also sends a snapshot on connect, but this avoids a loading flash on slow connections)
  const fetchQueueEntries = useCallback(async () => {
    try {
      setQueueLoading(true);
      const response = await GetAllDownloadQueueItems();
      if (response.status === "error") {
        setQueueError(response);
        return;
      }
      setJobs(response.data?.jobs || []);
      setQueueError(null);
    } finally {
      setQueueLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchQueueEntries();
    fetchHistory();
  }, [fetchQueueEntries, fetchHistory]);

  // Live updates over SSE, replacing the old 2s HTTP polling.
  useEffect(() => {
    const unsubscribe = subscribeToDownloadQueue({
      onSnapshot: (snapshotJobs) => {
        setJobs(snapshotJobs ?? []);
        setProgress(null);
        setQueueLoading(false);
        setQueueError(null);
      },
      onJobAdded: (job) => {
        // The snapshot query can race a real add, so the job may already be
        // in state by the time this event arrives - don't append a duplicate.
        setJobs((prev) => (prev.some((j) => j.id === job.id) ? prev : [...prev, job]));
      },
      onJobStarted: (job) => {
        setProgress(null);
        // The started job is processing, and is added if it was missing
        const started: DownloadQueueJob = { ...job, status: "processing" };
        setJobs((prev) =>
          prev.some((j) => j.id === job.id) ? prev.map((j) => (j.id === job.id ? started : j)) : [...prev, started]
        );
      },
      onJobProgress: (jobProgress) => {
        setProgress(jobProgress);
      },
      onJobFinished: (job) => {
        // Server always clears the job on finish, so drop it here too.
        setProgress((prev) => (prev?.job_id === job.id ? null : prev));
        setJobs((prev) => prev.filter((j) => j.id !== job.id));
        // A job finishing means a new history entry may exist.
        fetchHistory();
      },
      onJobRemoved: (job) => {
        setJobs((prev) => prev.filter((j) => j.id !== job.id));
      },
      onQueueCleared: () => {
        setJobs([]);
        setProgress(null);
      },
    });

    return unsubscribe;
  }, [fetchHistory]);

  const onClearQueue = async () => {
    const response = await RemoveAllFromQueue();
    if (response.status === "error") {
      toast.error(`Error clearing queue: ${response.error?.message || "Unknown error occurred."}`);
    } else {
      toast.success(response.data?.result || "Cleared the download queue.");
    }
  };

  const onClearHistory = async () => {
    const response = await RemoveAllDownloadHistory();
    if (response.status === "error") {
      toast.error(`Error clearing history: ${response.error?.message || "Unknown error occurred."}`);
    } else {
      toast.success(response.data?.result || "Cleared the download history.");
    }
    await fetchHistory();
  };

  // Finished jobs get recorded to History, so the queue only shows pending/processing.
  const activeJobs = jobs
    .filter((j) => j.status === "pending" || j.status === "processing")
    .sort((a, b) => (a.status === b.status ? 0 : a.status === "processing" ? -1 : 1));

  const processingJob = activeJobs.find((j) => j.status === "processing");

  return (
    <div className="container mx-auto p-4 min-h-screen flex flex-col items-center">
      <H2 className="text-3xl font-bold mb-4">Download Queue</H2>

      <Tabs defaultValue="queue" className="w-full">
        <TabsList className="mx-auto">
          <TabsTrigger value="queue">Queue</TabsTrigger>
          <TabsTrigger value="history">Failed</TabsTrigger>
        </TabsList>

        <TabsContent value="queue" className="w-full">
          {queueLoading ? (
            <Loader className="mt-10" message="Loading download queue entries..." />
          ) : queueError ? (
            <div className="flex flex-col items-center p-6 gap-4">
              <ErrorMessage error={queueError} />
            </div>
          ) : (
            <>
              <div className="w-full flex justify-end mb-2">
                <ConfirmDestructiveDialogActionButton
                  variant="outline"
                  className="text-destructive border-1 shadow-none hover:text-red-500 cursor-pointer"
                  confirmText="Clear Queue"
                  title="Clear Download Queue?"
                  description="Are you sure you want to remove all items from the download queue? Items currently downloading will not be interrupted. This action cannot be undone."
                  onConfirm={onClearQueue}
                  disabled={jobs.length === 0}
                >
                  Clear Queue
                </ConfirmDestructiveDialogActionButton>
              </div>

              {processingJob && (
                <DownloadQueueProgressPanel
                  progress={progress?.job_id === processingJob.id ? progress : null}
                  processingTitle={processingJob.media_item_title}
                  waiting={activeJobs.length - 1}
                />
              )}

              {activeJobs.length === 0 ? (
                <p className="text-gray-500">
                  No download queue entries found. Add some items to the queue to get started.
                </p>
              ) : (
                <ResponsiveGrid size="larger">
                  {activeJobs.map((job) => (
                    <DownloadEntryCard key={job.id} mode="queue" job={job} fetchQueueEntries={fetchQueueEntries} />
                  ))}
                </ResponsiveGrid>
              )}
            </>
          )}
        </TabsContent>

        <TabsContent value="history" className="w-full">
          {historyLoading ? (
            <Loader className="mt-10" message="Loading failed downloads..." />
          ) : historyError ? (
            <div className="flex flex-col items-center p-6 gap-4">
              <ErrorMessage error={historyError} />
            </div>
          ) : (
            <>
              <div className="w-full flex items-center justify-between gap-2 mb-2">
                <p className="text-sm text-muted-foreground">
                  Successful downloads show on the{" "}
                  <Link href="/saved-sets" className="text-primary hover:underline">
                    Saved Sets
                  </Link>{" "}
                  page.
                </p>
                <ConfirmDestructiveDialogActionButton
                  variant="outline"
                  className="text-destructive border-1 shadow-none hover:text-red-500 cursor-pointer"
                  confirmText="Clear Failed"
                  title="Clear Failed Downloads?"
                  description="Are you sure you want to remove all failed downloads from this list? This action cannot be undone."
                  onConfirm={onClearHistory}
                  disabled={historyEntries.length === 0}
                >
                  Clear Failed
                </ConfirmDestructiveDialogActionButton>
              </div>

              {historyEntries.length === 0 ? (
                <p className="text-gray-500">No failed downloads.</p>
              ) : (
                <ResponsiveGrid size="larger">
                  {historyEntries.map((entry) => (
                    <DownloadEntryCard key={entry.id} mode="history" entry={entry} fetchHistory={fetchHistory} />
                  ))}
                </ResponsiveGrid>
              )}
            </>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default DownloadQueuePage;
