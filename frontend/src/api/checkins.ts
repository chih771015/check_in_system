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

/** One diagnosis photo with its row ID, returned by the list endpoints. */
export interface DiagnosisPhotoItem {
  id: number;
  photoUrl: string;
}

/**
 * listDiagnosisPhotos returns the existing diagnosis photos (with IDs) for a
 * SchedulePatient owned by the current translator, so the manage modal can
 * preview and delete specific photos.
 */
export function listDiagnosisPhotos(schedulePatientId: number) {
  return client
    .get<{ photos: DiagnosisPhotoItem[] }>('/checkins/diagnosis/photos', {
      params: { schedulePatientId },
    })
    .then((r) => r.data.photos);
}

/** deleteDiagnosisPhoto removes one diagnosis photo owned by the translator. */
export function deleteDiagnosisPhoto(photoId: number) {
  return client.delete(`/checkins/diagnosis/photos/${photoId}`).then((r) => r.data);
}

/** Admin surrogate: list diagnosis photos for any SchedulePatient (with IDs). */
export function adminListDiagnosisPhotos(schedulePatientId: number) {
  return client
    .get<{ photos: DiagnosisPhotoItem[] }>('/admin/diagnosis/photos', {
      params: { schedulePatientId },
    })
    .then((r) => r.data.photos);
}

/** Admin surrogate: delete one diagnosis photo by ID. */
export function adminDeleteDiagnosisPhoto(photoId: number) {
  return client.delete(`/admin/diagnosis/photos/${photoId}`).then((r) => r.data);
}

/** setActualAmount records the translator's actual paid amount (整數元). */
export function setActualAmount(schedulePatientId: number, actualAmount: number) {
  return client
    .post('/checkins/diagnosis/amount', { schedulePatientId, actualAmount })
    .then((r) => r.data);
}

/** Admin surrogate: set actual paid amount for any SchedulePatient. */
export function adminSetActualAmount(schedulePatientId: number, actualAmount: number) {
  return client
    .post('/admin/diagnosis/amount', { schedulePatientId, actualAmount })
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
