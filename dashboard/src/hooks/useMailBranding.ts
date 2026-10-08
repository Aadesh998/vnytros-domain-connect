import { useCallback, useEffect, useState } from "react";
import { brandingClient } from "@/lib/mail/client";
import type { Branding, BrandingPayload } from "@/lib/mail/types";

export function useMailBranding() {
  const [branding, setBranding] = useState<Branding | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // State is only written from the promise callbacks, so the initial load can
  // run from an effect without a synchronous setState in the effect body.
  const load = useCallback(
    () =>
      brandingClient
        .get()
        .then(
          (data) => {
            setBranding(data);
            setError(null);
          },
          (err: unknown) => {
            setError(err instanceof Error ? err.message : "Failed to load branding");
          },
        )
        .finally(() => setIsLoading(false)),
    [],
  );

  const refresh = useCallback(() => {
    setIsLoading(true);
    setError(null);
    return load();
  }, [load]);

  useEffect(() => {
    void load();
  }, [load]);

  const update = async (payload: BrandingPayload) => {
    const updated = await brandingClient.update(payload);
    setBranding(updated);
    return updated;
  };

  return { branding, isLoading, error, refresh, update };
}
