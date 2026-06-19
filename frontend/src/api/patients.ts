import type {
  IDType,
  Patient,
  PatientHistoryResponse,
  PatientListResponse,
  TranslatorPatient,
} from '../types';
import client from './client';

export interface PatientPayload {
  name: string;
  phone: string;
  idType: IDType;
  idNumber: string;
}

export interface PatientListQuery {
  search?: string;
  page?: number;
  pageSize?: number;
}

export function getPatients(query: PatientListQuery = {}) {
  return client
    .get<PatientListResponse>('/admin/patients', { params: query })
    .then((r) => r.data);
}

export function createPatient(data: PatientPayload) {
  return client.post<{ data: Patient }>('/admin/patients', data).then((r) => r.data.data);
}

export function updatePatient(id: number, data: PatientPayload) {
  return client.put<{ data: Patient }>(`/admin/patients/${id}`, data).then((r) => r.data.data);
}

export function deletePatient(id: number) {
  return client.delete(`/admin/patients/${id}`).then((r) => r.data);
}

export function getPatientHistory(
  id: number,
  range?: { dateFrom?: string; dateTo?: string },
) {
  return client
    .get<PatientHistoryResponse>(`/admin/patients/${id}/history`, { params: range })
    .then((r) => r.data);
}

export interface PatientImportError {
  row: number;
  reason: string;
}

export interface PatientImportResult {
  created: number;
  skipped: number;
  errors: PatientImportError[];
}

/** Bulk-import patients from an xlsx file. Duplicates/invalid rows are skipped. */
export function importPatients(file: File) {
  const form = new FormData();
  form.append('file', file);
  return client
    .post<PatientImportResult>('/admin/patients/import', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data);
}

function downloadXlsx(path: string, filename: string) {
  return client.get(path, { responseType: 'blob' }).then((r) => {
    const url = URL.createObjectURL(new Blob([r.data]));
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  });
}

/** Download all patients as xlsx. */
export function exportPatients() {
  return downloadXlsx('/admin/export/patients', 'patients.xlsx');
}

/** Download the import template xlsx (header + example row). */
export function downloadPatientTemplate() {
  return downloadXlsx('/admin/export/patients-template', 'patients_template.xlsx');
}

export interface TranslatorPatientListResponse {
  data: TranslatorPatient[];
  total: number;
  page: number;
  pageSize: number;
}

// Translator-facing list. Stage 2 returns the same data set as admin (minus
// timestamps); stage 3 will scope the response to the caller's own schedules.
export function getPatientsForTranslator(query: PatientListQuery = {}) {
  return client
    .get<TranslatorPatientListResponse>('/patients', { params: query })
    .then((r) => r.data);
}
