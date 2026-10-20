"use client";

import { type CreatorSetCount, GetSavedSetCreators } from "@/services/database/creators";
import { DownloadFromFavoriteCreators } from "@/services/downloads/favorite-creators";
import { GetUserFollowingAndHiding } from "@/services/mediux/get-user-follow-hiding";
import { HandleSearch } from "@/services/search/api-search";
import { ArrowDown, ArrowUp, Plus, Star, X } from "lucide-react";
import { toast } from "sonner";

import { useEffect, useState } from "react";

import Link from "next/link";

import { CreatorAvatar } from "@/components/shared/creator-avatar";
import { ConfirmDestructiveDialogActionButton } from "@/components/shared/dialog-destructive-action";
import { PopoverHelp } from "@/components/shared/popover-help";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";

import { useOnboardingStore } from "@/lib/stores/global-store-onboarding";
import { useUserPreferencesStore } from "@/lib/stores/global-user-preferences";

import type { MediuxUserInfo } from "@/types/mediux/mediux-user-follow-hide";

// How long typing has to stop before asking for matching creators
const SUGGEST_DELAY_MS = 200;

const sameName = (a: string, b: string) => a.toLowerCase() === b.toLowerCase();

// Favorite creators ranked best first, shown when new items are downloaded automatically
export function FavoriteCreatorsCard() {
  const autoDownloadNewItems = useUserPreferencesStore((state) => state.autoDownloadNewItems);
  const setAutoDownloadNewItems = useUserPreferencesStore((state) => state.setAutoDownloadNewItems);
  const favoriteCreators = useUserPreferencesStore((state) => state.favoriteCreators);
  const setFavoriteCreators = useUserPreferencesStore((state) => state.setFavoriteCreators);
  const savedFavoriteCreators = useOnboardingStore(
    (state) => state.status?.current_setup?.user_preferences?.favorite_creators
  );

  // The creators followed on MediUX, as they are always favorites
  const [followed, setFollowed] = useState<string[]>([]);
  useEffect(() => {
    void GetUserFollowingAndHiding().then((response) =>
      setFollowed((response.data?.users ?? []).filter((user) => user.follow && !user.hide).map((user) => user.username))
    );
  }, []);
  const isFollowed = (username: string) => followed.some((name) => sameName(name, username));

  // Add followed creators last, once the list matches the saved one
  useEffect(() => {
    if (!autoDownloadNewItems || !savedFavoriteCreators) return;
    if (JSON.stringify(savedFavoriteCreators) !== JSON.stringify(favoriteCreators)) return;
    const missing = followed.filter((name) => !favoriteCreators.some((favorite) => sameName(favorite, name)));
    if (missing.length > 0) setFavoriteCreators([...favoriteCreators, ...missing]);
  }, [autoDownloadNewItems, followed, favoriteCreators, savedFavoriteCreators, setFavoriteCreators]);

  const [input, setInput] = useState("");
  const [suggestions, setSuggestions] = useState<MediuxUserInfo[]>([]);
  const [savedSetCreators, setSavedSetCreators] = useState<CreatorSetCount[]>([]);
  const [runningForAll, setRunningForAll] = useState(false);

  // Check the items that were already in the libraries
  const runForAllItems = async () => {
    setRunningForAll(true);
    const response = await DownloadFromFavoriteCreators();
    setRunningForAll(false);
    if (response.status === "error" || !response.data) {
      toast.error(response.error?.message || "Could not check the items");
      return;
    }
    const { checked, queued, no_set } = response.data;
    toast.success(
      `Checked ${checked} ${checked === 1 ? "item" : "items"} without a saved set. ${queued} added to the Download Queue, ${no_set} had no set from your favorite creators.`
    );
  };

  // The creators of the saved sets, to suggest as favorites
  useEffect(() => {
    void GetSavedSetCreators().then((response) => setSavedSetCreators(response.data?.creators ?? []));
  }, []);

  const isFavorite = (username: string) => favoriteCreators.some((favorite) => sameName(favorite, username));

  // Suggest MediUX creators while typing
  useEffect(() => {
    const query = input.trim();
    if (!query) return;
    const timer = setTimeout(async () => {
      const response = await HandleSearch({
        searchQuery: query,
        searchMediaItems: false,
        searchCollectionItems: false,
        searchMediuxUsers: true,
        searchSavedSets: false,
      });
      setSuggestions(response.data?.mediux_usernames ?? []);
    }, SUGGEST_DELAY_MS);
    return () => clearTimeout(timer);
  }, [input]);

  const shownSuggestions = input.trim() ? suggestions.filter((user) => !isFavorite(user.username)) : [];
  const savedSetSuggestions = savedSetCreators.filter((creator) => !isFavorite(creator.username));

  const add = (username: string) => {
    if (!username || isFavorite(username)) return;
    setFavoriteCreators([...favoriteCreators, username]);
    setInput("");
    setSuggestions([]);
  };

  // Only MediUX creators can be added, so a typed name has to match one
  const addTyped = () => {
    const match = shownSuggestions.find((user) => user.username.toLowerCase() === input.trim().toLowerCase());
    if (match) add(match.username);
  };

  const move = (index: number, by: number) => {
    const next = [...favoriteCreators];
    const [creator] = next.splice(index, 1);
    next.splice(index + by, 0, creator);
    setFavoriteCreators(next);
  };

  const remove = (index: number) => setFavoriteCreators(favoriteCreators.filter((_, i) => i !== index));

  return (
    <Card className="gap-4 border-muted p-4 sm:p-5">
      <h2 className="text-xl font-semibold">Auto Download New Items</h2>

      <div className="flex items-center justify-between gap-4">
        <Label htmlFor="auto-download-new-items" className="cursor-pointer">
          Automatically Download Images for New Items
        </Label>
        <div className="flex items-center gap-2">
          <Switch
            id="auto-download-new-items"
            checked={autoDownloadNewItems}
            onCheckedChange={setAutoDownloadNewItems}
            className="cursor-pointer"
          />
          <PopoverHelp ariaLabel="help-auto-download-new-items">
            <p className="mb-2">
              When a movie or show has no saved set, aura looks for a set by one of your favorite creators, then saves
              and downloads it.
            </p>
            <p className="mb-2">
              Your best ranked creator with a set wins. If they have more than one set, the most popular on MediUX is
              used. Items without a set from your favorite creators are skipped.
            </p>
            <p>The image types are your Download Defaults.</p>
          </PopoverHelp>
        </div>
      </div>

      {!autoDownloadNewItems ? (
        <p className="text-sm text-muted-foreground">
          Turn this on to choose the creators whose sets are downloaded for new items.
        </p>
      ) : (
        <>
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold">Favorite Creators</h3>
            <PopoverHelp ariaLabel="help-favorite-creators">
              <p className="mb-2">
                Your favorite MediUX creators, ranked best first. Rank 1 is your top creator. Use the arrows to change
                the order.
              </p>
              <p className="mb-2">
                Creators you follow on MediUX are always favorites. To remove one, unfollow them on MediUX.
              </p>
            </PopoverHelp>
          </div>

          {/* Add a creator */}
          <div className="relative">
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addTyped();
                }
              }}
              placeholder="Add a creator by MediUX username"
              aria-label="Add a favorite creator"
            />
            {shownSuggestions.length > 0 && (
              <div className="absolute z-20 mt-1 w-full overflow-hidden rounded-md border bg-background shadow-lg">
                {shownSuggestions.map((user) => (
                  <button
                    key={user.id || user.username}
                    type="button"
                    className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left text-sm hover:bg-muted"
                    onClick={() => add(user.username)}
                  >
                    <CreatorAvatar username={user.username} className="h-6 w-6" />
                    <span className="truncate">{user.username}</span>
                    {user.follow && <span className="ml-auto text-xs text-muted-foreground">Following</span>}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Ranked creators */}
          {favoriteCreators.length === 0 ? (
            <p className="text-sm text-muted-foreground">No favorite creators yet.</p>
          ) : (
            <ol className="divide-y rounded-md border">
              {favoriteCreators.map((username, index) => (
                <li key={username} className="flex items-center gap-2 px-2 py-2 sm:gap-3 sm:px-3">
                  <span className="w-4 shrink-0 text-right text-sm font-semibold text-muted-foreground sm:w-6">
                    {index + 1}
                  </span>
                  <CreatorAvatar username={username} className="h-8 w-8 shrink-0" />
                  <Link
                    href={`/user/${username}`}
                    className="min-w-0 truncate text-sm hover:text-primary hover:underline"
                  >
                    {username}
                  </Link>
                  {/* A star on narrow screens */}
                  {isFollowed(username) && (
                    <>
                      <Star
                        className="h-3.5 w-3.5 shrink-0 fill-current text-yellow-500 sm:hidden"
                        aria-label="Following"
                      />
                      <span className="hidden shrink-0 text-xs text-muted-foreground sm:inline">Following</span>
                    </>
                  )}
                  <div className="ml-auto flex shrink-0 items-center sm:gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 cursor-pointer sm:h-8 sm:w-8"
                      disabled={index === 0}
                      onClick={() => move(index, -1)}
                      aria-label={`Move ${username} up`}
                    >
                      <ArrowUp className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 cursor-pointer sm:h-8 sm:w-8"
                      disabled={index === favoriteCreators.length - 1}
                      onClick={() => move(index, 1)}
                      aria-label={`Move ${username} down`}
                    >
                      <ArrowDown className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 cursor-pointer text-destructive sm:h-8 sm:w-8"
                      disabled={isFollowed(username)}
                      title={
                        isFollowed(username) ? "Followed on MediUX. Unfollow them on MediUX to remove them" : undefined
                      }
                      onClick={() => remove(index)}
                      aria-label={`Remove ${username}`}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                </li>
              ))}
            </ol>
          )}

          {/* Suggestions from the saved sets */}
          {savedSetSuggestions.length > 0 && (
            <Accordion type="single" collapsible defaultValue="saved-set-creators" className="mt-3">
              <AccordionItem value="saved-set-creators" className="rounded-md border px-3">
                <AccordionTrigger className="cursor-pointer py-3 text-sm">
                  Suggestions from your saved sets ({savedSetSuggestions.length})
                </AccordionTrigger>
                <AccordionContent>
                  <ul className="divide-y">
                    {savedSetSuggestions.map((creator) => (
                      <li key={creator.username} className="flex items-center gap-3 py-2">
                        <CreatorAvatar username={creator.username} className="h-7 w-7 shrink-0" />
                        <Link
                          href={`/user/${creator.username}`}
                          className="min-w-0 truncate text-sm hover:text-primary hover:underline"
                        >
                          {creator.username}
                        </Link>
                        <span className="shrink-0 text-xs text-muted-foreground">
                          {creator.sets} saved {creator.sets === 1 ? "set" : "sets"}
                        </span>
                        <Button
                          variant="outline"
                          size="sm"
                          className="ml-auto h-7 shrink-0 cursor-pointer gap-1"
                          onClick={() => add(creator.username)}
                          aria-label={`Add ${creator.username} to favorite creators`}
                        >
                          <Plus className="h-3 w-3" />
                          Add
                        </Button>
                      </li>
                    ))}
                  </ul>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          )}

          {/* The items that were already in the libraries */}
          <div className="mt-5 flex flex-wrap items-center justify-between gap-3 rounded-md border p-3">
            <div className="min-w-0">
              <p className="text-sm font-medium">Items Already in Your Libraries</p>
              <p className="text-xs text-muted-foreground">
                New items are handled automatically. Run this to also check the items you already have.
              </p>
            </div>
            <ConfirmDestructiveDialogActionButton
              variant="outline"
              onConfirm={runForAllItems}
              disabled={runningForAll || favoriteCreators.length === 0}
              title="Run for All Items?"
              description="Every movie and show without a saved set gets the best set from your favorite creators, added to the Download Queue. On a large library this can be a lot of downloads."
              confirmText="Run for All Items"
            >
              {runningForAll ? "Checking..." : "Run for All Items"}
            </ConfirmDestructiveDialogActionButton>
          </div>
        </>
      )}
    </Card>
  );
}
