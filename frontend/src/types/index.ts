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

export interface ScheduleItem {
  id: number;
  translatorId: number;
  translatorName: string;
  date: string;
  startTime: string;
  endTime: string;
  location: string;
  patientName: string;
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
