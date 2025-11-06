import { useEffect, useState } from 'react';

interface SessionState {
  token: string | null;
  username: string | null;
  admin: boolean;
  setToken: (token: string | null, username?: string | null, admin?: boolean) => void;
}

type SessionSnapshot = {
  token: string | null;
  username: string | null;
  admin: boolean;
};

const tokenKey = 'ledger.session.token';
const usernameKey = 'ledger.session.username';
const adminKey = 'ledger.session.admin';

let snapshot: SessionSnapshot = { token: null, username: null, admin: false };
let hydrated = false;
const listeners = new Set<(value: SessionSnapshot) => void>();

const notify = () => {
  const payload = { ...snapshot };
  for (const listener of listeners) {
    listener(payload);
  }
};

const loadSnapshot = () => {
  if (hydrated || typeof window === 'undefined') {
    return snapshot;
  }
  snapshot = {
    token: window.localStorage.getItem(tokenKey),
    username: window.localStorage.getItem(usernameKey),
    admin: window.localStorage.getItem(adminKey) === 'true'
  };
  hydrated = true;
  return snapshot;
};

export const useSession = (): SessionState => {
  const [state, setState] = useState<SessionSnapshot>(() => ({ ...loadSnapshot() }));

  useEffect(() => {
    const handler = (value: SessionSnapshot) => setState({ ...value });
    listeners.add(handler);
    return () => {
      listeners.delete(handler);
    };
  }, []);

  const setToken = (token: string | null, username?: string | null, admin?: boolean) => {
    const current = loadSnapshot();
    const next: SessionSnapshot = {
      token,
      username: token ? username ?? current.username : null,
      admin: token ? (admin ?? current.admin) : false
    };
    snapshot = next;
    if (typeof window !== 'undefined') {
      if (token) {
        window.localStorage.setItem(tokenKey, token);
        if (next.username) {
          window.localStorage.setItem(usernameKey, next.username);
        } else {
          window.localStorage.removeItem(usernameKey);
        }
        window.localStorage.setItem(adminKey, next.admin ? 'true' : 'false');
      } else {
        window.localStorage.removeItem(tokenKey);
        window.localStorage.removeItem(usernameKey);
        window.localStorage.removeItem(adminKey);
      }
    }
    notify();
  };

  return { ...state, setToken };
};
