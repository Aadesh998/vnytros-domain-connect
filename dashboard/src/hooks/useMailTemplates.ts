import { useCallback, useEffect, useState } from "react";
import { mailTemplatesClient } from "@/lib/mail/client";
import type { MailTemplate, TemplatePayload } from "@/lib/mail/types";

export function useMailTemplates() {
  const [templates, setTemplates] = useState<MailTemplate[]>([]);
  const [drafts, setDrafts] = useState<MailTemplate[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // State is only written from the promise callbacks, so the initial load can
  // run from an effect without a synchronous setState in the effect body.
  const load = useCallback(
    () =>
      Promise.all([mailTemplatesClient.list(), mailTemplatesClient.listDrafts()])
        .then(
          ([published, draft]) => {
            setTemplates(published.templates ?? []);
            setDrafts(draft.templates ?? []);
            setError(null);
          },
          (err: unknown) => {
            setError(err instanceof Error ? err.message : "Failed to load templates");
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

  const create = async (payload: TemplatePayload) => {
    const created = await mailTemplatesClient.create(payload);
    await refresh();
    return created;
  };

  const update = async (id: number, payload: TemplatePayload) => {
    const updated = await mailTemplatesClient.update(id, payload);
    await refresh();
    return updated;
  };

  const remove = async (id: number) => {
    await mailTemplatesClient.delete(id);
    await refresh();
  };

  return {
    templates,
    drafts,
    /** Published and draft templates together, for selection lists. */
    all: [...templates, ...drafts],
    isLoading,
    error,
    refresh,
    create,
    update,
    remove,
  };
}
