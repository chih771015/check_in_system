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

export function getPatientHistory(id: number) {
  return client
    .get<PatientHistoryResponse>(`/admin/patients/${id}/history`)
    .then((r) => r.data);
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
