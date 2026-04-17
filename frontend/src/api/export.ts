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

export interface RunExportResult {
  format: string;
  emailTo: string;
  sheetUrl?: string;
  filename?: string;
  rangeFrom: string;
  rangeTo: string;
  ranAt: string;
}

export function runExportNow() {
  return client
    .post<{ message: string; result: RunExportResult }>('/admin/export/schedule/run')
    .then((r) => r.data);
}
