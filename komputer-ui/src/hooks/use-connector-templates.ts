"use client";

import { useState, useEffect } from "react";
import { listConnectorTemplates } from "@/lib/api";
import type { ConnectorTemplate } from "@/lib/types";

let cachedTemplates: ConnectorTemplate[] | null = null;

export function useConnectorTemplates() {
  const [templates, setTemplates] = useState<ConnectorTemplate[]>(cachedTemplates ?? []);
  const [loading, setLoading] = useState(cachedTemplates === null);

  useEffect(() => {
    if (cachedTemplates) return;
    listConnectorTemplates()
      .then((resp) => {
        cachedTemplates = resp.templates;
        setTemplates(resp.templates);
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const getByService = (service: string) =>
    templates.find((t) => t.service === service);

  return { templates, loading, getByService };
}
