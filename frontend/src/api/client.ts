import axios, { type AxiosError } from 'axios';
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

/**
 * unwrapResponse strips a `{ data: ... }` envelope when the body is just that
 * single key. Backend list endpoints wrap their payload this way.
 * Exported for testing.
 */
export function unwrapResponse<T>(data: T): T {
  if (
    data !== null &&
    typeof data === 'object' &&
    'data' in (data as object) &&
    Object.keys(data as object).length === 1
  ) {
    return (data as unknown as { data: T }).data;
  }
  return data;
}

/**
 * mapErrorResponse mutates an axios error so consumers receive the translated
 * message, and triggers global side-effects (redirect on 401 / forced password
 * change). Side-effects are intentionally side-effects so the same instance
 * works in production; tests can pass an alternative `navigate` mock.
 * Exported for testing.
 */
export function mapErrorResponse(
  error: AxiosError<{ code?: string; message?: string }>,
  redirect: (path: string) => void = (path) => {
    window.location.href = path;
  },
  currentPath: () => string = () => window.location.pathname,
): AxiosError & { translatedMessage?: string } {
  const status = error.response?.status;
  const code = error.response?.data?.code;
  const rawMessage = error.response?.data?.message;

  // Translate error code into i18n message; fall back to backend message.
  if (code) {
    const translated = i18n.t(`errors.${code}`, { defaultValue: rawMessage || '' });
    if (translated) {
      (error as AxiosError & { translatedMessage?: string }).translatedMessage = translated;
      if (error.response?.data) {
        error.response.data.message = translated;
      }
    }
  }

  if (status === 401) {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    redirect('/login');
  } else if (status === 403 && code === 'PASSWORD_CHANGE_REQUIRED') {
    if (currentPath() !== '/change-password') {
      redirect('/change-password');
    }
  }
  return error;
}

client.interceptors.response.use(
  (response) => {
    response.data = unwrapResponse(response.data);
    return response;
  },
  (error) => Promise.reject(mapErrorResponse(error as AxiosError<{ code?: string; message?: string }>)),
);

export default client;
