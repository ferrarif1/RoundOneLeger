import axios, { AxiosError } from 'axios';

// Determine API base URL based on current location or environment configuration.
// The previous implementation always targeted port 8080 which breaks when the
// frontend is served via HTTPS behind a reverse proxy (the browser refuses the
// https->http downgrade and reports a "Network Error").  Allow overriding via
// VITE_API_BASE_URL and only fall back to port 8080 for local development.
let baseURL = import.meta.env?.VITE_API_BASE_URL ?? '';

if (!baseURL && typeof window !== 'undefined') {
  const { hostname, protocol, port } = window.location;
  const isLocalhost = hostname === 'localhost' || hostname === '127.0.0.1';

  if (isLocalhost) {
    baseURL = `${protocol}//${hostname}:8080`;
  } else if (port && port !== '80' && port !== '443') {
    baseURL = `${protocol}//${hostname}:${port}`;
  } else {
    baseURL = `${protocol}//${hostname}`;
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