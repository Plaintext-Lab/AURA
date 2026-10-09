import { useOnboardingStore } from "@/lib/stores/global-store-onboarding";

// Media servers that do not support collections
const COLLECTIONS_NOT_SUPPORTED_TYPES = ["Silo"];

export function useCollectionsSupported(): boolean {
  const mediaServerType = useOnboardingStore((state) => state.status?.current_setup?.media_server?.type);
  return !COLLECTIONS_NOT_SUPPORTED_TYPES.includes(mediaServerType ?? "");
}
