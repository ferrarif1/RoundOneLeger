import { useEffect, useState } from 'react';

interface SessionState {
  token: string | null;
  setToken: (token: string | null) => void;
}

const sessionKey = 'ledger.session.token';
let sessionToken: string | null = null;
const listeners = new Set<(value: string | null) => void>();

const notify = () => {
  for (const listener of listeners) {
    listener(sessionToken);
  }
};

const ensureTokenLoaded = () => {
  if (sessionToken === null) {
    sessionToken = window.localStorage.getItem(sessionKey);
  }
  return sessionToken;
};

export const useSession = (): SessionState => {
  const [token, setTokenState] = useState<string | null>(() => ensureTokenLoaded());

  useEffect(() => {
    const handler = (value: string | null) => setTokenState(value);
    listeners.add(handler);
    return () => {
      listeners.delete(handler);
    };
  }, []);

  const setToken = (next: string | null) => {
    sessionToken = next;
    if (next) {
      window.localStorage.setItem(sessionKey, next);
    } else {
      window.localStorage.removeItem(sessionKey);
    }
    notify();
  };

  return { token, setToken };
};
