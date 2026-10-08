import { useCallback, useEffect, useState } from "react";
import { domainsClient } from "@/lib/domains/client";
import type { Domain } from "@/lib/domains/types";

export function useDomains() {
  const [domains, setDomains] = useState<Domain[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // State is only written from the promise callbacks, so the initial load can
  // run from an effect without a synchronous setState in the effect body.
  const load = useCallback(
    () =>
      domainsClient
        .list()
        .then(
          (data) => {
            setDomains(Array.isArray(data) ? data : []);
            setError(null);
          },
          (err: unknown) => {
            setError(err instanceof Error ? err.message : "Failed to load domains");
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

  const verify = useCallback(
    async (domain: string) => {
      const res = await domainsClient.verify(domain);
      await refresh();
      return res;
    },
    [refresh],
  );

  useEffect(() => {
    void load();
  }, [load]);

  return { domains, isLoading, error, refresh, verify };
}
