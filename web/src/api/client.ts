import axios, { AxiosError } from 'axios';

// Determine API base URL based on current location or environment configuration.
// Prefer staying on the current origin when possible to keep HTTPS deployments
// secure, but fall back to the typical Go API port (8080) when the frontend is
// being accessed through an IP/hostname without an explicit port.
let baseURL = import.meta.env?.VITE_API_BASE_URL ?? '';

if (!baseURL && typeof window !== 'undefined') {
  const { hostname, protocol, port } = window.location;
  const normalizedPort = port || (protocol === 'https:' ? '443' : protocol === 'http:' ? '80' : '');
  const isDefaultPort = normalizedPort === '80' || normalizedPort === '443' || normalizedPort === '';
  const isViteDevPort = normalizedPort === '5173' || normalizedPort === '4173';

  if (isViteDevPort) {
    // Let the Vite proxy rewrite requests to the backend when running the dev server.
    baseURL = '';
  } else if (!isDefaultPort) {
    baseURL = `${protocol}//${hostname}:${port}`;
  } else if (import.meta.env.DEV) {
    // Dev builds without an explicit port still need to talk to the Go server on 8080.
    baseURL = `${protocol}//${hostname}:8080`;
  } else {
    // Production deployments commonly front the API on the same origin.
    baseURL = `${protocol}//${hostname}:8080`;
  }
}

// If the heuristics above still left baseURL empty (e.g. SSR), fall back to relative requests.
if (!baseURL) {
  baseURL = '';
}

const api = axios.create({
  baseURL,
  withCredentials: true,
});

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('ledger.token');
    if (token) {
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
