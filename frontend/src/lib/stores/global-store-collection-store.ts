import type { CollectionItem } from "@/app/collections/page";
import { create } from "zustand";
import { persist } from "zustand/middleware";

import { GlobalStore } from "@/lib/stores/stores";

interface CollectionStore {
  collectionItem: CollectionItem | null;
  setCollectionItem: (collectionItem: CollectionItem | null) => void;

  clear: () => void;
  hasHydrated: boolean;
  hydrate: () => void;
}

export const useCollectionStore = create<CollectionStore>()(
  persist(
    (set) => ({
      collectionItem: null,
      setCollectionItem: (collectionItem) => set({ collectionItem }),

      clear: () => set({ collectionItem: null }),
      hasHydrated: false,
      hydrate: () => set({ hasHydrated: true }),
    }),
    {
      name: "CurrentCollection",
      storage: GlobalStore,
      partialize: (state) => ({
        collectionItem: state.collectionItem,
      }),
      onRehydrateStorage: () => (state) => {
        state?.hydrate();
      },
    }
  )
);
