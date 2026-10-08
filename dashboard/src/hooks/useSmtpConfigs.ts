import { useCallback, useEffect, useState } from "react";
import { smtpConfigsClient } from "@/lib/mail/client";
import type { SmtpConfig, SmtpConfigPayload } from "@/lib/mail/types";

export function useSmtpConfigs() {
  const [configs, setConfigs] = useState<SmtpConfig[]>([]);
  const [defaultId, setDefaultId] = useState<number | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // State is only written from the promise callbacks, so the initial load can
  // run from an effect without a synchronous setState in the effect body.
  const load = useCallback(
    () =>
      smtpConfigsClient
        .list()
        .then(
          (data) => {
            setConfigs(data.configs ?? []);
            setDefaultId(data.default_id);
            setError(null);
          },
          (err: unknown) => {
            setError(err instanceof Error ? err.message : "Failed to load senders");
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

  const create = async (payload: SmtpConfigPayload) => {
    const created = await smtpConfigsClient.create(payload);
    await refresh();
    return created;
  };

  const update = async (id: number, payload: SmtpConfigPayload) => {
    const updated = await smtpConfigsClient.update(id, payload);
    await refresh();
    return updated;
  };

  const setDefault = async (id: number) => {
    await smtpConfigsClient.setDefault(id);
    await refresh();
  };

  const remove = async (id: number) => {
    await smtpConfigsClient.delete(id);
    await refresh();
  };

  const seed = async () => {
    const seeded = await smtpConfigsClient.seed();
    await refresh();
    return seeded;
  };

  return {
    configs,
    defaultId,
    isLoading,
    error,
    refresh,
    create,
    update,
    setDefault,
    remove,
    seed,
  };
}
