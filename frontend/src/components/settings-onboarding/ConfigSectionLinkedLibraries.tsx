"use client";

import { ReturnErrorMessage } from "@/services/api-error-return";
import {
  CreateLibraryGroup,
  DeleteLibraryGroup,
  GetLibraryGroups,
  PreviewLibraryGroup,
  ReconcileLibraryGroup,
  UpdateLibraryGroup,
} from "@/services/database/library-groups";
import { AlertCircle, CheckCircle2, Loader2, Plus, RefreshCw, Trash2, X } from "lucide-react";
import { toast } from "sonner";

import React, { useCallback, useEffect, useState } from "react";

import { ConfirmDestructiveDialogActionButton } from "@/components/shared/dialog-destructive-action";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import type { AppConfigMediaServerLibrary } from "@/types/config/config";
import type {
  LibraryGroup,
  LibraryGroupPreview,
  LibraryGroupPreviewItem,
  ReconcileDecision,
} from "@/types/database/library-group";

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

interface ConfigSectionLinkedLibrariesProps {
  /** Available libraries from the media server config. */
  availableLibraries: AppConfigMediaServerLibrary[];
}

interface GroupFormState {
  name: string;
  media_type: "movie" | "show" | "";
  library_ids: string[];
}

const emptyForm = (): GroupFormState => ({ name: "", media_type: "", library_ids: [] });

// ─────────────────────────────────────────────────────────────────────────────
// Component
// ─────────────────────────────────────────────────────────────────────────────

export const ConfigSectionLinkedLibraries: React.FC<ConfigSectionLinkedLibrariesProps> = ({
  availableLibraries,
}) => {
  const [groups, setGroups] = useState<LibraryGroup[]>([]);
  const [loading, setLoading] = useState(true);

  // Dialog state
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<LibraryGroup | null>(null);
  const [form, setForm] = useState<GroupFormState>(emptyForm());
  const [formError, setFormError] = useState<string>("");

  // Preview state
  const [preview, setPreview] = useState<LibraryGroupPreview | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);

  // Reconcile state
  const [reconcileOpen, setReconcileOpen] = useState(false);
  const [reconcileGroup, setReconcileGroup] = useState<LibraryGroup | null>(null);
  const [reconcileDecisions, setReconcileDecisions] = useState<Record<string, ReconcileDecision>>(
    {}
  );
  const [reconciling, setReconciling] = useState(false);

  // ─── Load groups ─────────────────────────────────────────────────────────

  const loadGroups = useCallback(async () => {
    setLoading(true);
    try {
      const res = await GetLibraryGroups();
      if (res.status === "error") {
        toast.error(res.error?.message || "Failed to load library groups");
      } else {
        setGroups(res.data?.groups ?? []);
      }
    } catch (err) {
      const r = ReturnErrorMessage<never>(err);
      toast.error(r.error?.message || "Failed to load library groups");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadGroups();
  }, [loadGroups]);

  // ─── Helpers ─────────────────────────────────────────────────────────────

  const compatibleLibraries = availableLibraries.filter(
    (lib) => !form.media_type || lib.type === form.media_type
  );

  const libraryName = (id: string) =>
    availableLibraries.find((l) => l.id === id)?.title ?? id;

  // ─── Dialog open/close ───────────────────────────────────────────────────

  const openCreate = () => {
    setEditingGroup(null);
    setForm(emptyForm());
    setFormError("");
    setPreview(null);
    setDialogOpen(true);
  };

  const openEdit = (group: LibraryGroup) => {
    setEditingGroup(group);
    setForm({
      name: group.name,
      media_type: group.media_type,
      library_ids: group.library_ids,
    });
    setFormError("");
    setPreview(null);
    setDialogOpen(true);
  };

  const closeDialog = () => {
    setDialogOpen(false);
    setPreview(null);
    setFormError("");
  };

  // ─── Preview ─────────────────────────────────────────────────────────────

  const handlePreview = async () => {
    if (!form.media_type || form.library_ids.length < 2) {
      setFormError("Select a media type and at least two libraries before previewing.");
      return;
    }
    setFormError("");
    setPreviewLoading(true);
    try {
      const res = await PreviewLibraryGroup(editingGroup?.id ?? "new", {
        name: form.name || "preview",
        media_type: form.media_type,
        library_ids: form.library_ids,
      });
      if (res.status === "error") {
        toast.error(res.error?.message || "Preview failed");
      } else {
        setPreview(res.data?.preview ?? null);
      }
    } catch (err) {
      const r = ReturnErrorMessage<never>(err);
      toast.error(r.error?.message || "Preview failed");
    } finally {
      setPreviewLoading(false);
    }
  };

  // ─── Save ────────────────────────────────────────────────────────────────

  const handleSave = async () => {
    if (!form.name.trim()) {
      setFormError("Group name is required.");
      return;
    }
    if (!form.media_type) {
      setFormError("Media type is required.");
      return;
    }
    if (form.library_ids.length < 2) {
      setFormError("Select at least two libraries.");
      return;
    }
    setFormError("");
    try {
      const payload = {
        name: form.name.trim(),
        media_type: form.media_type,
        library_ids: form.library_ids,
      };
      const res = editingGroup
        ? await UpdateLibraryGroup(editingGroup.id, payload)
        : await CreateLibraryGroup(payload);

      if (res.status === "error") {
        toast.error(res.error?.message || "Failed to save group");
        return;
      }
      toast.success(editingGroup ? "Group updated." : "Group created.");
      closeDialog();
      void loadGroups();
    } catch (err) {
      const r = ReturnErrorMessage<never>(err);
      toast.error(r.error?.message || "Failed to save group");
    }
  };

  // ─── Delete ──────────────────────────────────────────────────────────────

  const handleDelete = async (group: LibraryGroup) => {
    try {
      const res = await DeleteLibraryGroup(group.id);
      if (res.status === "error") {
        toast.error(res.error?.message || "Failed to delete group");
        return;
      }
      toast.success(`Group "${group.name}" deleted. Existing applied artwork is preserved.`);
      void loadGroups();
    } catch (err) {
      const r = ReturnErrorMessage<never>(err);
      toast.error(r.error?.message || "Failed to delete group");
    }
  };

  // ─── Reconcile ───────────────────────────────────────────────────────────

  const openReconcile = async (group: LibraryGroup) => {
    setReconcileGroup(group);
    setReconcileDecisions({});
    setReconcileOpen(true);
    // Fetch a fresh preview for this group
    setPreviewLoading(true);
    try {
      const res = await PreviewLibraryGroup(group.id, {
        name: group.name,
        media_type: group.media_type,
        library_ids: group.library_ids,
      });
      if (res.status === "error") {
        toast.error(res.error?.message || "Failed to load reconciliation preview");
      } else {
        setPreview(res.data?.preview ?? null);
        // Pre-populate decisions for non-conflicting items
        const decisions: Record<string, ReconcileDecision> = {};
        for (const item of res.data?.preview?.items ?? []) {
          if (!item.has_conflict) {
            const firstSet = item.library_sets[0]?.saved_sets[0];
            if (firstSet) {
              decisions[`${item.tmdb_id}|${item.edition}`] = {
                set_id: firstSet.id,
                selected_types: firstSet.selected_types,
                auto_download: false,
              };
            }
          }
        }
        setReconcileDecisions(decisions);
      }
    } catch (err) {
      const r = ReturnErrorMessage<never>(err);
      toast.error(r.error?.message || "Failed to load reconciliation preview");
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleReconcile = async () => {
    if (!reconcileGroup) return;
    setReconciling(true);
    try {
      const res = await ReconcileLibraryGroup(reconcileGroup.id, {
        policy_decisions: reconcileDecisions,
      });
      if (res.status === "error") {
        toast.error(res.error?.message || "Reconciliation failed");
        return;
      }
      toast.success(
        `Reconciled ${res.data?.reconciled ?? 0} items, skipped ${res.data?.skipped ?? 0}.`
      );
      setReconcileOpen(false);
      void loadGroups();
    } catch (err) {
      const r = ReturnErrorMessage<never>(err);
      toast.error(r.error?.message || "Reconciliation failed");
    } finally {
      setReconciling(false);
    }
  };

  // ─── Render ──────────────────────────────────────────────────────────────

  return (
    <>
      <Card className="p-5 border-muted">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-blue-500">Linked Libraries</h2>
          <Button size="sm" variant="outline" onClick={openCreate}>
            <Plus className="w-4 h-4 mr-1" />
            New Group
          </Button>
        </div>

        <p className="text-sm text-muted-foreground mb-4">
          Link compatible libraries (e.g. <em>Movies</em> and <em>Movies 4K</em>) so that selecting
          artwork once applies it consistently across all copies.
        </p>

        {loading ? (
          <div className="flex items-center gap-2 text-muted-foreground text-sm">
            <Loader2 className="w-4 h-4 animate-spin" />
            Loading groups…
          </div>
        ) : groups.length === 0 ? (
          <p className="text-sm text-muted-foreground italic">No linked library groups yet.</p>
        ) : (
          <div className="space-y-3">
            {groups.map((group) => (
              <div
                key={group.id}
                className="flex items-start justify-between rounded-md border border-muted p-3"
              >
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{group.name}</span>
                    <Badge variant="secondary" className="text-xs capitalize">
                      {group.media_type}
                    </Badge>
                  </div>
                  <div className="flex flex-wrap gap-1">
                    {group.library_ids.map((id) => (
                      <Badge key={id} variant="outline" className="text-xs">
                        {libraryName(id)}
                      </Badge>
                    ))}
                  </div>
                </div>
                <div className="flex items-center gap-1 ml-2 shrink-0">
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => openReconcile(group)}
                    title="Reconcile"
                  >
                    <RefreshCw className="w-4 h-4" />
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => openEdit(group)}
                    title="Edit"
                  >
                    Edit
                  </Button>
                  <ConfirmDestructiveDialogActionButton
                    title={`Delete group "${group.name}"?`}
                    description="Existing applied artwork will be preserved. Future artwork changes will no longer be shared across these libraries."
                    onConfirm={() => handleDelete(group)}
                  >
                    <Button size="sm" variant="ghost" className="text-destructive" title="Delete">
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </ConfirmDestructiveDialogActionButton>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Create / Edit dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{editingGroup ? "Edit Library Group" : "Create Library Group"}</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 pt-2">
            {/* Name */}
            <div className="space-y-1">
              <Label>Group Name</Label>
              <Input
                placeholder="e.g. Movies (All Versions)"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </div>

            {/* Media Type */}
            <div className="space-y-1">
              <Label>Media Type</Label>
              <Select
                value={form.media_type}
                onValueChange={(v) =>
                  setForm((f) => ({
                    ...f,
                    media_type: v as "movie" | "show",
                    library_ids: [],
                  }))
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select media type…" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="movie">Movie</SelectItem>
                  <SelectItem value="show">Show</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Libraries */}
            <div className="space-y-2">
              <Label>Libraries (minimum 2)</Label>
              {compatibleLibraries.length === 0 && form.media_type && (
                <p className="text-xs text-muted-foreground">
                  No compatible {form.media_type} libraries found in your configuration.
                </p>
              )}
              <div className="grid grid-cols-1 gap-1">
                {compatibleLibraries.map((lib) => {
                  const selected = form.library_ids.includes(lib.id);
                  return (
                    <label
                      key={lib.id}
                      className={`flex items-center gap-2 rounded border px-3 py-2 cursor-pointer text-sm transition ${
                        selected
                          ? "border-blue-500 bg-blue-500/10"
                          : "border-muted hover:border-muted-foreground"
                      }`}
                    >
                      <input
                        type="checkbox"
                        checked={selected}
                        onChange={() =>
                          setForm((f) => ({
                            ...f,
                            library_ids: selected
                              ? f.library_ids.filter((id) => id !== lib.id)
                              : [...f.library_ids, lib.id],
                          }))
                        }
                        className="sr-only"
                      />
                      <span className={selected ? "text-blue-400" : ""}>{lib.title}</span>
                      {lib.id && (
                        <span className="ml-auto text-xs text-muted-foreground font-mono">
                          {lib.id}
                        </span>
                      )}
                    </label>
                  );
                })}
              </div>
            </div>

            {formError && (
              <p className="text-sm text-red-500 flex items-center gap-1">
                <AlertCircle className="w-4 h-4" />
                {formError}
              </p>
            )}

            {/* Preview */}
            <Button
              variant="outline"
              size="sm"
              onClick={handlePreview}
              disabled={previewLoading || form.library_ids.length < 2}
              className="w-full"
            >
              {previewLoading ? (
                <Loader2 className="w-4 h-4 animate-spin mr-1" />
              ) : (
                <RefreshCw className="w-4 h-4 mr-1" />
              )}
              Preview Matches &amp; Conflicts
            </Button>

            {preview && <PreviewSummary preview={preview} />}

            <div className="flex justify-end gap-2 pt-2">
              <Button variant="ghost" onClick={closeDialog}>
                Cancel
              </Button>
              <Button onClick={handleSave}>
                {editingGroup ? "Save Changes" : "Create Group"}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Reconcile dialog */}
      <Dialog open={reconcileOpen} onOpenChange={setReconcileOpen}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Reconcile: {reconcileGroup?.name}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Items with conflicting artwork selections require a decision. Choose a set to use for
            each conflict, or skip the item to leave it unlinked.
          </p>

          {previewLoading ? (
            <div className="flex items-center gap-2 text-muted-foreground text-sm py-4">
              <Loader2 className="w-4 h-4 animate-spin" />
              Loading preview…
            </div>
          ) : preview && preview.items.length > 0 ? (
            <div className="space-y-3 mt-2">
              {preview.items
                .filter((item) => item.has_conflict)
                .map((item) => (
                  <ConflictItemRow
                    key={`${item.tmdb_id}|${item.edition}`}
                    item={item}
                    decision={reconcileDecisions[`${item.tmdb_id}|${item.edition}`]}
                    onDecide={(d) =>
                      setReconcileDecisions((prev) => ({
                        ...prev,
                        [`${item.tmdb_id}|${item.edition}`]: d,
                      }))
                    }
                    onSkip={() =>
                      setReconcileDecisions((prev) => {
                        const next = { ...prev };
                        delete next[`${item.tmdb_id}|${item.edition}`];
                        return next;
                      })
                    }
                  />
                ))}
              {preview.conflict_count === 0 && (
                <p className="text-sm text-green-500 flex items-center gap-1">
                  <CheckCircle2 className="w-4 h-4" />
                  No conflicts detected. All {preview.match_count} matched items will be linked.
                </p>
              )}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground italic py-2">No matching items found.</p>
          )}

          <div className="flex justify-end gap-2 pt-4">
            <Button variant="ghost" onClick={() => setReconcileOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleReconcile} disabled={reconciling}>
              {reconciling && <Loader2 className="w-4 h-4 animate-spin mr-1" />}
              Apply Decisions
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
};

// ─────────────────────────────────────────────────────────────────────────────
// Sub-components
// ─────────────────────────────────────────────────────────────────────────────

function PreviewSummary({ preview }: { preview: LibraryGroupPreview }) {
  return (
    <div className="rounded-md border border-muted p-3 text-sm space-y-1">
      <p>
        <span className="font-medium">{preview.match_count}</span> matching item
        {preview.match_count !== 1 ? "s" : ""} found.
      </p>
      {preview.conflict_count > 0 ? (
        <p className="text-amber-500 flex items-center gap-1">
          <AlertCircle className="w-4 h-4" />
          {preview.conflict_count} item{preview.conflict_count !== 1 ? "s" : ""} have conflicting
          saved sets and require reconciliation.
        </p>
      ) : (
        preview.match_count > 0 && (
          <p className="text-green-500 flex items-center gap-1">
            <CheckCircle2 className="w-4 h-4" />
            No conflicts detected.
          </p>
        )
      )}
    </div>
  );
}

function ConflictItemRow({
  item,
  decision,
  onDecide,
  onSkip,
}: {
  item: LibraryGroupPreviewItem;
  decision: ReconcileDecision | undefined;
  onDecide: (d: ReconcileDecision) => void;
  onSkip: () => void;
}) {
  const allSets = item.library_sets.flatMap((ls) =>
    ls.saved_sets.map((s) => ({ ...s, libraryTitle: ls.library_title }))
  );
  const uniqueSets = allSets.filter(
    (s, i, arr) => arr.findIndex((x) => x.id === s.id) === i
  );

  return (
    <div className="rounded-md border border-amber-500/40 p-3 space-y-2 text-sm">
      <div className="flex items-center justify-between">
        <span className="font-medium">
          {item.title} ({item.year})
          {item.edition && (
            <span className="ml-1 text-muted-foreground text-xs">– {item.edition}</span>
          )}
        </span>
        <Badge variant="secondary" className="text-xs">
          conflict
        </Badge>
      </div>
      <div className="space-y-1">
        {item.library_sets.map((ls) => (
          <div key={ls.library_id} className="flex items-center gap-2 text-xs text-muted-foreground">
            <span className="font-medium">{ls.library_title}:</span>
            {ls.saved_sets.length === 0 ? (
              <span className="italic">no saved set</span>
            ) : (
              ls.saved_sets.map((s) => (
                <Badge key={s.id} variant="outline">
                  {s.id}
                </Badge>
              ))
            )}
          </div>
        ))}
      </div>
      <div className="flex items-center gap-2 pt-1">
        <span className="text-xs text-muted-foreground">Use set:</span>
        <Select
          value={decision?.set_id ?? ""}
          onValueChange={(v) =>
            onDecide({
              set_id: v,
              selected_types: uniqueSets.find((s) => s.id === v)?.selected_types ?? {
                poster: true,
                backdrop: false,
                season_poster: false,
                special_season_poster: false,
                titlecard: false,
              },
              auto_download: false,
            })
          }
        >
          <SelectTrigger className="h-7 text-xs flex-1">
            <SelectValue placeholder="Select a set…" />
          </SelectTrigger>
          <SelectContent>
            {uniqueSets.map((s) => (
              <SelectItem key={s.id} value={s.id}>
                {s.id}{" "}
                <span className="text-muted-foreground ml-1">({s.libraryTitle})</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          size="sm"
          variant="ghost"
          className="h-7 px-2 text-xs text-muted-foreground"
          onClick={onSkip}
          title="Skip – leave unlinked"
        >
          <X className="w-3 h-3 mr-0.5" />
          Skip
        </Button>
      </div>
    </div>
  );
}
