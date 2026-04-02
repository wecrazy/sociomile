import { useEffect, useState } from "react";
import { apiRequest } from "./api";
import type { User } from "./auth";

type Agent = Pick<User, "id" | "name">;

export function useTenantAgents(token: string | null) {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (!token) {
        setAgents([]);
        return;
      }

      setLoading(true);
      try {
        const result = await apiRequest<Agent[]>("/users/agents", { token });
        if (!cancelled) {
          setAgents(result.data);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    load().catch(() => undefined);

    return () => {
      cancelled = true;
    };
  }, [token]);

  return { agents, loading };
}
