import axios, { AxiosError } from 'axios';

// Determine API base URL based on current location
let baseURL = '';
if (typeof window !== 'undefined') {
  const { hostname, protocol } = window.location;
  // For localhost, API is on port 8080
  if (hostname === 'localhost' || hostname === '127.0.0.1') {
    baseURL = `${protocol}//${hostname}:8080`;
  } else {
    // For other hosts, API is on the same host but port 8080
    baseURL = `${protocol}//${hostname}:8080`;
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