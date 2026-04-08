import type { CheckinItem } from '../types';
import client from './client';

export function checkin(formData: FormData) {
  return client
    .post('/checkins', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data);
}

export function makeupCheckin(formData: FormData) {
  return client
    .post('/checkins/makeup', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data);
}

export function getAdminCheckins(params?: Record<string, string>) {
  return client
    .get<CheckinItem[]>('/admin/checkins', { params })
    .then((r) => r.data);
}

export function exportCheckinGoogleSheet(params?: Record<string, string>) {
  return client
    .post<{ url: string; title: string }>('/admin/export/google-sheet', {}, { params })
    .then((r) => r.data);
}

export function exportCheckinExcel(params?: Record<string, string>) {
  return client
    .get('/admin/export/excel', {
      params,
      responseType: 'blob',
    })
    .then((r) => {
      const url = URL.createObjectURL(new Blob([r.data]));
      const a = document.createElement('a');
      a.href = url;
      a.download = 'checkins.xlsx';
      a.click();
      URL.revokeObjectURL(url);
    });
}
