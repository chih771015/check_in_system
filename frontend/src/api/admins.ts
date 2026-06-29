import type { AdminListItem } from '../types';
import client from './client';

export interface AdminListResponse {
  data: AdminListItem[];
  total: number;
  page: number;
  pageSize: number;
}

// getAdmins returns one page plus the total. Omit page/pageSize to get every row.
export function getAdmins(params?: { page?: number; pageSize?: number }) {
  return client.get<AdminListResponse>('/admin/admins', { params }).then((r) => r.data);
}

export function createAdmin(data: { email: string; name: string; password: string }) {
  return client.post('/admin/admins', data).then((r) => r.data);
}

export function deleteAdmin(id: number) {
  return client.delete(`/admin/admins/${id}`).then((r) => r.data);
}
