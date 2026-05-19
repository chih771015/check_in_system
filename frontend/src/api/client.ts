import axios from 'axios';
import i18n from '../i18n';

const client = axios.create({
  // 使用相對路徑，讓 nginx（Docker）或 Vite dev proxy 統一轉發
  // 避免 baseURL 寫死 localhost 導致外部瀏覽器打到本機
  baseURL: '/api',
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
    const rawMessage = error.response?.data?.message;

    // Translate error code into i18n message; fallback to backend message
    if (code) {
      const translated = i18n.t(`errors.${code}`, { defaultValue: rawMessage || '' });
      if (translated) {
        error.translatedMessage = translated;
        // Override the response message so callers using err.response.data.message
        // get the translated version automatically.
        if (error.response?.data) {
          error.response.data.message = translated;
        }
      }
    }

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
