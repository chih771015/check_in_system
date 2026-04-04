import type { ScheduleItem } from '../types';
import client from './client';

export function getAdminSchedules(params?: Record<string, string>) {
  return client.get<ScheduleItem[]>('/admin/schedules', { params }).then((r) => r.data);
}

export function createSchedule(data: {
  translatorId: number;
  date: string;
  startTime: string;
  endTime: string;
  location: string;
  patientName: string;
  note?: string;
}) {
  return client.post('/admin/schedules', data).then((r) => r.data);
}

export function updateSchedule(
  id: number,
  data: {
    translatorId?: number;
    date?: string;
    startTime?: string;
    endTime?: string;
    location?: string;
    patientName?: string;
    note?: string;
  },
) {
  return client.put(`/admin/schedules/${id}`, data).then((r) => r.data);
}

export function deleteSchedule(id: number) {
  return client.delete(`/admin/schedules/${id}`).then((r) => r.data);
}

export function getMySchedules(params?: Record<string, string>) {
  return client.get<ScheduleItem[]>('/schedules', { params }).then((r) => r.data);
}
