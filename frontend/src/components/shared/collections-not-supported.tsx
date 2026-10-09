"use client";

import { ReturnErrorMessage } from "@/services/api-error-return";

import { useRouter } from "next/navigation";

import { ErrorMessage } from "@/components/shared/error-message";
import { Button } from "@/components/ui/button";

import { useOnboardingStore } from "@/lib/stores/global-store-onboarding";

export function CollectionsNotSupported() {
  const router = useRouter();
  const mediaServerType = useOnboardingStore((state) => state.status?.current_setup?.media_server?.type);

  return (
    <div className="flex flex-col items-center">
      <ErrorMessage
        error={ReturnErrorMessage<string>(
          `Collections are not supported for ${mediaServerType || "this Media Server"}`
        )}
        isWarning={true}
      />
      <Button
        className="mt-4"
        variant="secondary"
        onClick={() => {
          router.push("/");
        }}
      >
        Go to Home
      </Button>
    </div>
  );
}
