export interface User {
  id: number;
  email: string;
  name: string;
  phone: string;
  role: 'admin' | 'translator';
  status: 'active' | 'disabled';
  mustChangePW: boolean;
}

export interface TranslatorListItem {
  id: number;
  email: string;
  name: string;
  phone: string;
  status: 'active' | 'disabled';
  createdAt: string;
}

export type SchedulePatientStatus = 'pending' | 'completed' | 'no_show';

export interface SchedulePatient {
  id: number;
  patientId: number;
  patientName: string;
  patientPhone: string;
  idType: IDType;
  idNumber: string;
  startTime: string;
  endTime: string;
  status: SchedulePatientStatus;
  noShowReason?: string;
  prepaidAmount: number;
  actualAmount: number;
}

export interface DiagnosisPhoto {
  id: number;
  schedulePatientId: number;
  photoUrl: string;
  uploadedAt: string;
}

/** Row in the admin "diagnosis results overview" table. */
export interface DiagnosisResult {
  schedulePatientId: number;
  scheduleId: number;
  date: string;
  startTime: string;
  endTime: string;
  location: string;
  note: string;
  translatorId: number;
  translatorName: string;
  patientId: number;
  patientName: string;
  patientPhone: string;
  idType: IDType;
  idNumber: string;
  status: 'completed' | 'no_show';
  noShowReason?: string;
  diagnosisPhotos: string[];
  prepaidAmount: number;
  actualAmount: number;
  updatedAt: string;
}

export interface DiagnosisResultsResponse {
  data: DiagnosisResult[];
  total: number;
  page: number;
  pageSize: number;
}

export interface SchedulePatientPayload {
  patientId: number;
  startTime: string;
  endTime: string;
  prepaidAmount: number;
}

export interface ScheduleItem {
  id: number;
  translatorId: number;
  translatorName: string;
  date: string;
  startTime: string;
  endTime: string;
  location: string;
  /** Legacy single-patient name (stage 1/2 data). Empty/absent for new schedules. */
  patientName: string;
  patients: SchedulePatient[];
  note: string;
  checkinStatus: 'none' | 'arrived' | 'completed' | 'makeup';
  recurrenceGroupId?: string | null;
}

export interface CheckinItem {
  id: number;
  scheduleId: number;
  translatorId: number;
  translatorName: string;
  type: 'arrive' | 'leave';
  checkinTime: string;
  latitude: number;
  longitude: number;
  address: string;
  selfieUrl: string;
  environmentUrl: string;
  isMakeup: boolean;
  makeupReason: string;
  createdAt: string;
}

export interface AdminListItem {
  id: number;
  email: string;
  name: string;
  status: 'active' | 'disabled';
  createdAt: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface ApiError {
  code: string;
  message: string;
}

export type IDType = 'passport' | 'hn' | 'unid';

export interface Patient {
  id: number;
  name: string;
  phone: string;
  idType: IDType;
  idNumber: string;
  createdAt: string;
  updatedAt: string;
}

export interface TranslatorPatient {
  id: number;
  name: string;
  phone: string;
  idType: IDType;
  idNumber: string;
}

export interface PatientHistoryEntry {
  scheduleId: number;
  date: string;
  startTime: string;
  endTime: string;
  location: string;
  translatorName: string;
  status: string;
  noShowReason?: string;
  diagnosisPhotos: string[];
  prepaidAmount: number;
  actualAmount: number;
}

export interface PatientListResponse {
  data: Patient[];
  total: number;
  page: number;
  pageSize: number;
}

export interface PatientHistoryResponse {
  patient: Patient;
  history: PatientHistoryEntry[];
}
