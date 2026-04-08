import client from './client';

export interface ExportScheduleData {
  frequency: string;
  dayOfMonth: number;
  format: string;
  emailTo: string;
  enabled: boolean;
  lastRunAt?: string;
}

export function getExportSchedule() {
  return client.get<ExportScheduleData>('/admin/export/schedule').then((r) => r.data);
}

export function upsertExportSchedule(data: Omit<ExportScheduleData, 'lastRunAt'>) {
  return client.post('/admin/export/schedule', data).then((r) => r.data);
}
