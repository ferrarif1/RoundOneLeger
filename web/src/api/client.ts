import axios, { AxiosError } from 'axios';

const resolveBaseURL = (): string | undefined => {
  const envBase = import.meta.env.VITE_API_BASE_URL?.trim();
  if (envBase) {
    return envBase;
  }

  if (typeof window === 'undefined') {
    return undefined;
  }

  const { origin, port, protocol, hostname } = window.location;

  // In development we rely on the Vite proxy, so use relative URLs.
  if (import.meta.env.DEV) {
    return undefined;
  }

  if (origin === 'null' || origin.startsWith('file:')) {
    return 'http://127.0.0.1:8080';
  }

  if (protocol === 'http:' || protocol === 'https:') {
    if (port && port !== '80' && port !== '443') {
      return `${protocol}//${hostname}:${port}`;
    }
    return origin;
  }

  return undefined;
};

const api = axios.create({
  baseURL: resolveBaseURL(),
  withCredentials: true,
});

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('ledger.token') : null;
    if (token) {
      config.headers = config.headers ?? {};
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor to handle errors
api.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    // Return error for component to handle
    return Promise.reject(error);
  }
);

export default api;
