import { createContext, useContext, useMemo, useState, type ReactNode } from "react";
import { apiRequest } from "./api";

type Tenant = {
  id: string;
  name: string;
  slug?: string;
};

export type User = {
  id: string;
  tenant_id: string;
  name: string;
  email: string;
  role: string;
  tenant?: Tenant;
};

type LoginResponse = {
  access_token: string;
  user: User;
};

type AuthContextValue = {
  token: string | null;
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

const STORAGE_KEY = "sociomile-auth";

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState(() => {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return { token: null as string | null, user: null as User | null };
    }

    try {
      return JSON.parse(raw) as { token: string | null; user: User | null };
    } catch {
      return { token: null as string | null, user: null as User | null };
    }
  });

  const value = useMemo<AuthContextValue>(
    () => ({
      token: state.token,
      user: state.user,
      login: async (email: string, password: string) => {
        const result = await apiRequest<LoginResponse>("/auth/login", {
          method: "POST",
          body: { email, password },
        });

        const nextState = { token: result.data.access_token, user: result.data.user };
        window.localStorage.setItem(STORAGE_KEY, JSON.stringify(nextState));
        setState(nextState);
      },
      logout: () => {
        window.localStorage.removeItem(STORAGE_KEY);
        setState({ token: null, user: null });
      },
    }),
    [state.token, state.user],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }

  return context;
}
