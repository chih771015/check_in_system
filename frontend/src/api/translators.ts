import type { TranslatorListItem } from '../types';
import client from './client';

export function getTranslators(status?: string) {
  return client
    .get<TranslatorListItem[]>('/admin/translators', { params: status ? { status } : undefined })
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
