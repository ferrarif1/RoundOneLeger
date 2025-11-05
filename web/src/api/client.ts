import axios, { AxiosError } from 'axios';

const envBaseURL = import.meta.env?.VITE_API_BASE_URL?.trim() ?? '';

let baseURL = envBaseURL;

if (!baseURL && typeof window !== 'undefined') {
  const { hostname, protocol, port } = window.location;
  const normalizedPort = port || (protocol === 'https:' ? '443' : protocol === 'http:' ? '80' : '');
  const isDefaultPort = normalizedPort === '80' || normalizedPort === '443' || normalizedPort === '';
  const isViteDevPort = normalizedPort === '5173' || normalizedPort === '4173';
  const isLoopbackHost =
    hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1';

  if (isViteDevPort) {
    // Let the Vite dev/preview servers proxy API traffic so that remote devices
    // reuse the same origin (http://<host>:5173) instead of hitting the backend
    // directly and tripping CORS or network restrictions.
    baseURL = '';
  } else if (!isDefaultPort) {
    baseURL = `${protocol}//${hostname}:${port}`;
  } else if (isLoopbackHost) {
    // Loopback development should default to the Go server's port.
    baseURL = `${protocol}//${hostname}:8080`;
  } else {
    // For named hosts keep calls on the current origin so that reverse proxies
    // (e.g. Nginx serving both frontend + backend) continue to work without extra config.
    baseURL = `${protocol}//${hostname}`;
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
