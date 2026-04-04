import type { LoginResponse } from '../types';
import client from './client';

export function login(email: string, password: string) {
  return client.post<LoginResponse>('/auth/login', { email, password }).then((r) => r.data);
}

export function changePassword(oldPassword: string, newPassword: string) {
  return client.post('/auth/change-password', { oldPassword, newPassword }).then((r) => r.data);
}
