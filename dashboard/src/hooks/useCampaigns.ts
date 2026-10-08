import { useCallback, useEffect, useState } from "react";
import { campaignsClient } from "@/lib/mail/client";
import type { Campaign, CampaignPayload } from "@/lib/mail/types";

/** Statuses whose counters are still moving, so the list should keep polling. */
const LIVE_STATUSES = new Set(["queued", "in_progress"]);
const POLL_INTERVAL_MS = 5000;

export function useCampaigns(scope: "sent" | "drafts" = "sent") {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextId, setNextId] = useState<number | undefined>(undefined);
  const [hasMore, setHasMore] = useState(false);

  const fetchPage = useCallback(
    (lastId?: number) =>
      scope === "drafts"
        ? campaignsClient.listDrafts(lastId)
        : campaignsClient.list(lastId),
    [scope],
  );

  const [loadedScope, setLoadedScope] = useState(scope);
  if (loadedScope !== scope) {
    setLoadedScope(scope);
    setIsLoading(true);
    setError(null);
  }

  // State is only written from the promise callbacks, so the load can run
  // from an effect without a synchronous setState in the effect body.
  const load = useCallback(
    () =>
      fetchPage(undefined)
        .then(
          (page) => {
            const items = page.campaigns ?? [];
            setCampaigns(items);
            setNextId(page.next_id);
            setHasMore(items.length > 0 && page.next_id !== undefined);
            setError(null);
          },
          (err: unknown) => {
            setError(err instanceof Error ? err.message : "Failed to load campaigns");
          },
        )
        .finally(() => setIsLoading(false)),
    [fetchPage],
  );

  const refresh = useCallback(() => {
    setIsLoading(true);
    setError(null);
    return load();
  }, [load]);

  // Silent refresh used by the poller so in-flight campaigns tick upward
  // without flashing the loading state.
  const refreshQuietly = useCallback(async () => {
    try {
      const page = await fetchPage(undefined);
      setCampaigns(page.campaigns ?? []);
      setNextId(page.next_id);
    } catch {
      // A failed background poll is not worth surfacing; the next tick retries.
    }
  }, [fetchPage]);

  const loadMore = useCallback(async () => {
    if (!hasMore || nextId === undefined || isLoading) return;
    setIsLoading(true);
    try {
      const page = await fetchPage(nextId);
      const items = page.campaigns ?? [];
      setCampaigns((prev) => [...prev, ...items]);
      setNextId(page.next_id);
      setHasMore(items.length > 0 && page.next_id !== undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load more");
    } finally {
      setIsLoading(false);
    }
  }, [fetchPage, hasMore, nextId, isLoading]);

  useEffect(() => {
    void load();
  }, [load]);

  const hasLive = campaigns.some((c) => LIVE_STATUSES.has(c.status));

  useEffect(() => {
    if (!hasLive) return;
    const timer = setInterval(() => void refreshQuietly(), POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [hasLive, refreshQuietly]);

  const create = async (payload: CampaignPayload) => {
    const created = await campaignsClient.create(payload);
    await refresh();
    return created;
  };

  const update = async (id: number, payload: CampaignPayload) => {
    const updated = await campaignsClient.update(id, payload);
    await refresh();
    return updated;
  };

  const remove = async (id: number) => {
    await campaignsClient.delete(id);
    await refresh();
  };

  const send = async (id: number, audience: File, smtpConfigId?: number) => {
    const result = await campaignsClient.send(id, audience, smtpConfigId);
    await refresh();
    return result;
  };

  return {
    campaigns,
    isLoading,
    error,
    hasMore,
    loadMore,
    refresh,
    create,
    update,
    remove,
    send,
  };
}
