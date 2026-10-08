import { useCallback, useEffect, useState } from "react";
import { campaignsClient, mailAnalyticsClient } from "@/lib/mail/client";
import type { AccountAnalytics, CampaignAnalytics } from "@/lib/mail/types";

/** Account-wide mail overview. */
export function useMailAnalytics(days = 30) {
  const [analytics, setAnalytics] = useState<AccountAnalytics | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [loadedDays, setLoadedDays] = useState(days);

  // A new window is a new load: flip to loading during render rather than
  // from the effect that fetches it.
  if (loadedDays !== days) {
    setLoadedDays(days);
    setIsLoading(true);
    setError(null);
  }

  // State is only written from the promise callbacks, so the load can run
  // from an effect without a synchronous setState in the effect body.
  const load = useCallback(
    () =>
      mailAnalyticsClient
        .overview(days)
        .then(
          (data) => {
            setAnalytics(data);
            setError(null);
          },
          (err: unknown) => {
            setError(err instanceof Error ? err.message : "Failed to load analytics");
          },
        )
        .finally(() => setIsLoading(false)),
    [days],
  );

  const refresh = useCallback(() => {
    setIsLoading(true);
    setError(null);
    return load();
  }, [load]);

  useEffect(() => {
    void load();
  }, [load]);

  return { analytics, isLoading, error, refresh };
}

const LIVE_STATUSES = new Set(["queued", "in_progress"]);
const POLL_INTERVAL_MS = 5000;

/**
 * Per-campaign report. Polls while the campaign is still sending so progress
 * and open counts advance on screen.
 */
export function useCampaignAnalytics(campaignId: number | null, days = 30) {
  const [analytics, setAnalytics] = useState<CampaignAnalytics | null>(null);
  const [isLoading, setIsLoading] = useState(campaignId !== null);
  const [error, setError] = useState<string | null>(null);
  const [loadedKey, setLoadedKey] = useState(`${campaignId}:${days}`);

  const key = `${campaignId}:${days}`;
  if (loadedKey !== key) {
    setLoadedKey(key);
    setIsLoading(campaignId !== null);
    setError(null);
  }

  // State is only written from the promise callbacks, so the load can run
  // from an effect without a synchronous setState in the effect body.
  const load = useCallback(() => {
    if (campaignId === null) return Promise.resolve();
    return campaignsClient
      .analytics(campaignId, days)
      .then(
        (data) => {
          setAnalytics(data);
          setError(null);
        },
        (err: unknown) => {
          setError(err instanceof Error ? err.message : "Failed to load report");
        },
      )
      .finally(() => setIsLoading(false));
  }, [campaignId, days]);

  const refresh = useCallback(() => {
    if (campaignId === null) return Promise.resolve();
    setIsLoading(true);
    setError(null);
    return load();
  }, [campaignId, load]);

  const refreshQuietly = useCallback(async () => {
    if (campaignId === null) return;
    try {
      setAnalytics(await campaignsClient.analytics(campaignId, days));
    } catch {
      // Background poll; the next tick retries.
    }
  }, [campaignId, days]);

  useEffect(() => {
    void load();
  }, [load]);

  const isLive = analytics ? LIVE_STATUSES.has(analytics.status) : false;

  useEffect(() => {
    if (!isLive) return;
    const timer = setInterval(() => void refreshQuietly(), POLL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [isLive, refreshQuietly]);

  return { analytics, isLoading, error, refresh };
}
