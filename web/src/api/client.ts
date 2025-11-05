import axios, { AxiosError } from 'axios';

// Determine API base URL based on current location or environment configuration.
// Prefer staying on the current origin (which keeps HTTPS deployments secure and
// allows the Vite dev server proxy to handle cross-origin requests) unless an
// explicit override is provided via VITE_API_BASE_URL.
let baseURL = import.meta.env?.VITE_API_BASE_URL ?? '';

if (!baseURL && typeof window !== 'undefined') {
  const { hostname, protocol, port } = window.location;
  const normalizedPort = port || (protocol === 'https:' ? '443' : protocol === 'http:' ? '80' : '');
  const isDefaultPort = normalizedPort === '80' || normalizedPort === '443' || normalizedPort === '';
  const isViteDevPort = normalizedPort === '5173' || normalizedPort === '4173';

  if (isViteDevPort) {
    // Let the Vite proxy rewrite requests to the backend.
    baseURL = '';
  } else if (isDefaultPort) {
    baseURL = `${protocol}//${hostname}`;
  } else {
    baseURL = `${protocol}//${hostname}:${port}`;
  }
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
