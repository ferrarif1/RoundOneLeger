import axios from 'axios';

const resolveBaseURL = (): string | undefined => {
  const explicit = import.meta.env.VITE_API_URL;
  if (explicit && explicit.trim()) {
    return explicit.trim();
  }
  if (typeof window !== 'undefined') {
    return window.location.origin;
  }
  return undefined;
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
