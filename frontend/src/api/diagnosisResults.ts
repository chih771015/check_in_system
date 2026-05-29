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
