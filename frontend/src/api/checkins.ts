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

export function getMyCheckins(params?: Record<string, string>) {
  return client
    .get<CheckinItem[]>('/checkins', { params })
    .then((r) => r.data);
}

export interface MyCheckinStats {
  total: number;
  arriveCount: number;
  leaveCount: number;
  makeupCount: number;
  onTimeCount: number;
  lateCount: number;
}

export function getMyCheckinStats(params?: Record<string, string>) {
  return client
    .get<MyCheckinStats>('/checkins/stats', { params })
    .then((r) => r.data);
}

export function updateCheckin(
  id: number,
  data: { checkinTime?: string; address?: string; makeupReason?: string },
) {
  return client.put(`/admin/checkins/${id}`, data).then((r) => r.data);
}

export function deleteCheckin(id: number) {
  return client.delete(`/admin/checkins/${id}`).then((r) => r.data);
}

export function exportCheckinGoogleSheet(params?: Record<string, string>) {
  return client
    .post<{ url: string; title: string }>('/admin/export/google-sheet', {}, { params })
    .then((r) => r.data);
}

/**
 * uploadDiagnosis appends 1..3 diagnosis photos to a SchedulePatient.
 * After upload the slot's status flips to "completed".
 */
export function uploadDiagnosis(schedulePatientId: number, photos: File[]) {
  const form = new FormData();
  form.append('schedulePatientId', String(schedulePatientId));
  photos.forEach((p) => form.append('photo', p));
  return client
    .post<{ message: string; photoUrls: string[] }>('/checkins/diagnosis', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data);
}

/** markNoShow flips a SchedulePatient's status to no_show with a reason. */
export function markNoShow(schedulePatientId: number, reason: string) {
  return client
    .post('/checkins/no-show', { schedulePatientId, reason })
    .then((r) => r.data);
}

/** Admin surrogate: same as uploadDiagnosis but without ownership check. */
export function adminUploadDiagnosis(schedulePatientId: number, photos: File[]) {
  const form = new FormData();
  form.append('schedulePatientId', String(schedulePatientId));
  photos.forEach((p) => form.append('photo', p));
  return client
    .post<{ message: string }>('/admin/diagnosis', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data);
}

/** Admin surrogate: same as markNoShow but without ownership check. */
export function adminMarkNoShow(schedulePatientId: number, reason: string) {
  return client
    .post('/admin/no-show', { schedulePatientId, reason })
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
