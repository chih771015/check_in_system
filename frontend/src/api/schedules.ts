import type { ScheduleItem, SchedulePatientPayload } from '../types';
import client from './client';

export interface AdminScheduleListResponse {
  data: ScheduleItem[];
  total: number;
  page: number;
  pageSize: number;
}

export function getAdminSchedules(params?: Record<string, string | number>) {
  return client
    .get<AdminScheduleListResponse>('/admin/schedules', { params })
    .then((r) => r.data);
}

export interface CreateSchedulePayload {
  translatorId: number;
  date: string;
  startTime: string;
  endTime: string;
  location: string;
  /** New stage-3 multi-patient field. When provided takes precedence over patientName. */
  patients?: SchedulePatientPayload[];
  /** Legacy single-patient field. Kept for backward compat. */
  patientName?: string;
  note?: string;
  recurrenceRule?: string;
  recurrenceUntil?: string;
}

export function createSchedule(data: CreateSchedulePayload) {
  return client.post('/admin/schedules', data).then((r) => r.data);
}

export interface UpdateSchedulePayload {
  translatorId?: number;
  date?: string;
  startTime?: string;
  endTime?: string;
  location?: string;
  patients?: SchedulePatientPayload[];
  patientName?: string;
  note?: string;
}

export function updateSchedule(id: number, data: UpdateSchedulePayload) {
  return client.put(`/admin/schedules/${id}`, data).then((r) => r.data);
}

export function deleteSchedule(id: number) {
  return client.delete(`/admin/schedules/${id}`).then((r) => r.data);
}

export function deleteScheduleGroup(id: number) {
  return client.delete(`/admin/schedules/${id}/group`).then((r) => r.data);
}

export interface ImportFailedRow {
  rowNumber: number;
  code?: string;
  error: string;
}

export interface ImportResult {
  total: number;
  successSchedules: number;
  successPatients: number;
  failed: ImportFailedRow[];
}

export function importSchedules(file: File) {
  const form = new FormData();
  form.append('file', file);
  return client
    .post<ImportResult>('/admin/schedules/import', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data);
}

export function getMySchedules(params?: Record<string, string>) {
  return client.get<ScheduleItem[]>('/schedules', { params }).then((r) => r.data);
}
