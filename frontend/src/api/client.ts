import axios from 'axios';

const client = axios.create({
  baseURL: 'http://localhost:8080/api',
});

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

client.interceptors.response.use(
  (response) => {
    // Unwrap { data: [...] } envelope from backend list responses
    if (
      response.data !== null &&
      typeof response.data === 'object' &&
      'data' in response.data &&
      Object.keys(response.data).length === 1
    ) {
      response.data = response.data.data;
    }
    return response;
  },
  (error) => {
    const status = error.response?.status;
    const code = error.response?.data?.code;
    if (status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    } else if (status === 403 && code === 'PASSWORD_CHANGE_REQUIRED') {
      if (window.location.pathname !== '/change-password') {
        window.location.href = '/change-password';
      }
    }
    return Promise.reject(error);
  },
);

export default client;
