import type { TranslatorListItem } from '../types';
import client from './client';

export interface TranslatorListResponse {
  data: TranslatorListItem[];
  total: number;
  page: number;
  pageSize: number;
}

// getTranslators returns the FULL list (no pagination) — used by the translator
// dropdown pickers across the admin UI. The list endpoint always returns the
// paginated envelope; omitting page/pageSize makes the backend return every row.
export function getTranslators(status?: string) {
  return client
    .get<TranslatorListResponse>('/admin/translators', { params: status ? { status } : undefined })
    .then((r) => r.data.data);
}

// getTranslatorsPaged returns one page plus the total, for the management table.
export function getTranslatorsPaged(params: { status?: string; page: number; pageSize: number }) {
  const { status, page, pageSize } = params;
  return client
    .get<TranslatorListResponse>('/admin/translators', {
      params: { ...(status ? { status } : {}), page, pageSize },
    })
    .then((r) => r.data);
}

export function createTranslator(data: {
  name: string;
  email: string;
  phone: string;
  password: string;
}) {
  return client.post('/admin/translators', data).then((r) => r.data);
}

export function updateTranslator(
  id: number,
  data: { name?: string; phone?: string; status?: string },
) {
  return client.put(`/admin/translators/${id}`, data).then((r) => r.data);
}

export function disableTranslator(id: number) {
  return client.delete(`/admin/translators/${id}`).then((r) => r.data);
}

export function resetTranslatorPassword(id: number, newPassword: string) {
  return client
    .post(`/admin/translators/${id}/reset-password`, { newPassword })
    .then((r) => r.data);
}
