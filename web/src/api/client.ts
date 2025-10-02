import axios from 'axios';

const DEV_PORTS = new Set(['5173', '5174', '4173', '4174']);

const resolveBaseURL = (): string => {
  const explicit = import.meta.env.VITE_API_URL;
  if (explicit && explicit.trim()) {
    return explicit.trim();
  }

  if (typeof window !== 'undefined') {
    const { protocol, hostname, port } = window.location;

    const isHttp = protocol === 'http:' || protocol === 'https:';

    if (!isHttp || !hostname) {
      return 'http://localhost:8080';
    }

    if (port && DEV_PORTS.has(port)) {
      return `${protocol}//${hostname}:8080`;
    }

    if (port) {
      return `${protocol}//${hostname}:${port}`;
    }

    return `${protocol}//${hostname}`;
  }

  return 'http://localhost:8080';
};

const api = axios.create({
  baseURL: resolveBaseURL(),
  withCredentials: true
});

api.interceptors.request.use((config) => {
  const token = window.localStorage.getItem('ledger.session.token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export default api;
