import { create } from "zustand";
import { persist } from "zustand/middleware";

import { GlobalStore } from "@/lib/stores/stores";

import { DOWNLOAD_IMAGE_TYPE_OPTIONS, type TYPE_DOWNLOAD_IMAGE_TYPE_OPTIONS } from "@/types/ui-options";

interface UserPreferencesStore {
  downloadDefaults: TYPE_DOWNLOAD_IMAGE_TYPE_OPTIONS[];
  setDownloadDefaults: (downloadDefaults: TYPE_DOWNLOAD_IMAGE_TYPE_OPTIONS[]) => void;

  showOnlyDownloadDefaults: boolean;
  setShowOnlyDownloadDefaults: (showOnlyDownloadDefaults: boolean) => void;

  showDateModified: boolean;
  setShowDateModified: (showDateModified: boolean) => void;

  defaultAddToQueue: boolean;
  setDefaultAddToQueue: (defaultAddToQueue: boolean) => void;

  hasHydrated: boolean;
  hydrate: () => void;
  clear: () => void;
}

export const useUserPreferencesStore = create<UserPreferencesStore>()(
  persist(
    (set) => ({
      downloadDefaults: DOWNLOAD_IMAGE_TYPE_OPTIONS.map((option) => option.value),
      setDownloadDefaults: (downloadDefaults: TYPE_DOWNLOAD_IMAGE_TYPE_OPTIONS[]) => set({ downloadDefaults }),

      showOnlyDownloadDefaults: false,
      setShowOnlyDownloadDefaults: (showOnlyDownloadDefaults: boolean) => set({ showOnlyDownloadDefaults }),

      showDateModified: false,
      setShowDateModified: (showDateModified: boolean) => set({ showDateModified }),

      defaultAddToQueue: false,
      setDefaultAddToQueue: (defaultAddToQueue: boolean) => set({ defaultAddToQueue }),

      hasHydrated: false,
      hydrate: () => set({ hasHydrated: true }),

      clear: () =>
        set({
          downloadDefaults: DOWNLOAD_IMAGE_TYPE_OPTIONS.map((option) => option.value),
          showOnlyDownloadDefaults: false,
          showDateModified: false,
          defaultAddToQueue: false,
        }),
    }),
    {
      name: "UserPreferences",
      storage: GlobalStore,
      partialize: (state) => ({
        downloadDefaults: state.downloadDefaults,
        showOnlyDownloadDefaults: state.showOnlyDownloadDefaults,
        showDateModified: state.showDateModified,
        defaultAddToQueue: state.defaultAddToQueue,
      }),
      onRehydrateStorage: () => (state) => {
        state?.hydrate();
      },
    }
  )
);
