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
