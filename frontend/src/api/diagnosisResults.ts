import type { DiagnosisResultsResponse } from '../types';
import client from './client';

export interface DiagnosisResultsQuery {
  dateFrom?: string;
  dateTo?: string;
  translatorId?: number;
  patientName?: string;
  status?: 'completed' | 'no_show';
  page?: number;
  pageSize?: number;
}

export function getDiagnosisResults(q: DiagnosisResultsQuery) {
  return client
    .get<DiagnosisResultsResponse>('/admin/diagnosis-results', { params: q })
    .then((r) => r.data);
}

/** Download the diagnosis-results overview (with amounts) as xlsx, same filters. */
export function exportDiagnosisResults(q: DiagnosisResultsQuery) {
  return client
    .get('/admin/export/diagnosis', { params: q, responseType: 'blob' })
    .then((r) => {
      const url = URL.createObjectURL(new Blob([r.data]));
      const a = document.createElement('a');
      a.href = url;
      a.download = 'diagnosis_results.xlsx';
      a.click();
      URL.revokeObjectURL(url);
    });
}

/** Fetch the diagnosis photo URLs for a single SchedulePatient slot. */
export function getSchedulePatientPhotos(schedulePatientId: number) {
  return client
    .get<{ photos: string[] }>(`/admin/schedule-patients/${schedulePatientId}/photos`)
    .then((r) => r.data.photos);
}
