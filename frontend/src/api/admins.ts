import type { AdminListItem } from '../types';
import client from './client';

export function getAdmins() {
  return client.get<{ data: AdminListItem[] }>('/admin/admins').then((r) => r.data.data);
}

export function createAdmin(data: { email: string; name: string; password: string }) {
  return client.post('/admin/admins', data).then((r) => r.data);
}

export function deleteAdmin(id: number) {
  return client.delete(`/admin/admins/${id}`).then((r) => r.data);
}
