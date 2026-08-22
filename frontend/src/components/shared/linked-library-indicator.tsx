"use client";

import { GetLibraryGroups } from "@/services/database/library-groups";

import { useEffect, useState } from "react";

import { Badge } from "@/components/ui/badge";

import type { LibraryGroup } from "@/types/database/library-group";

interface LinkedLibraryIndicatorProps {
  /** The stable library ID of the current media item's library. */
  libraryId: string;
}

/**
 * Shows a small status indicator when the current library belongs to one or
 * more linked-library groups, listing all the other linked library titles.
 */
export function LinkedLibraryIndicator({ libraryId }: LinkedLibraryIndicatorProps) {
  const [linkedGroups, setLinkedGroups] = useState<LibraryGroup[]>([]);

  useEffect(() => {
    if (!libraryId) return;
    let cancelled = false;
    GetLibraryGroups().then((res) => {
      if (cancelled || res.status === "error") return;
      const groups = (res.data?.groups ?? []).filter((g) =>
        g.library_ids.includes(libraryId)
      );
      setLinkedGroups(groups);
    });
    return () => {
      cancelled = true;
    };
  }, [libraryId]);

  if (linkedGroups.length === 0) return null;

  return (
    <div className="flex flex-wrap items-center gap-1 mt-1">
      {linkedGroups.map((group) => {
        const otherIds = group.library_ids.filter((id) => id !== libraryId);
        return (
          <Badge
            key={group.id}
            variant="outline"
            className="text-xs border-blue-500 text-blue-400 gap-1"
            title={`Linked group: ${group.name}`}
          >
            🔗{" "}
            {otherIds.length > 0
              ? `Linked (${group.name})`
              : group.name}
          </Badge>
        );
      })}
    </div>
  );
}
