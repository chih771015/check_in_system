import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import type { ReactNode } from 'react';
import { AuthProvider, useAuth } from '../authStore';
import type { User } from '../../types';

const sampleUser: User = {
  id: 1,
  email: 'admin@a.com',
  name: 'Admin',
  phone: '',
  role: 'admin',
  status: 'active',
  mustChangePW: false,
};

function wrapper({ children }: { children: ReactNode }) {
  return <AuthProvider>{children}</AuthProvider>;
}

describe('authStore', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('starts logged out when localStorage is empty', () => {
    const { result } = renderHook(() => useAuth(), { wrapper });
    expect(result.current.user).toBeNull();
    expect(result.current.token).toBeNull();
    expect(result.current.isAdmin).toBe(false);
    expect(result.current.isTranslator).toBe(false);
  });

  it('restores user + token from localStorage on init', () => {
    localStorage.setItem('token', 'persisted-tok');
    localStorage.setItem('user', JSON.stringify(sampleUser));

    const { result } = renderHook(() => useAuth(), { wrapper });
    expect(result.current.token).toBe('persisted-tok');
    expect(result.current.user?.email).toBe('admin@a.com');
    expect(result.current.isAdmin).toBe(true);
  });

  it('login() writes to localStorage and updates state', () => {
    const { result } = renderHook(() => useAuth(), { wrapper });

    act(() => {
      result.current.login('tk', sampleUser);
    });

    expect(result.current.token).toBe('tk');
    expect(result.current.user?.id).toBe(1);
    expect(localStorage.getItem('token')).toBe('tk');
    expect(JSON.parse(localStorage.getItem('user') ?? '{}').email).toBe('admin@a.com');
  });

  it('logout() clears state and localStorage', () => {
    localStorage.setItem('token', 'tk');
    localStorage.setItem('user', JSON.stringify(sampleUser));
    const { result } = renderHook(() => useAuth(), { wrapper });

    act(() => {
      result.current.logout();
    });

    expect(result.current.token).toBeNull();
    expect(result.current.user).toBeNull();
    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('user')).toBeNull();
  });

  it('updateUser() merges partial updates and persists', () => {
    localStorage.setItem('token', 'tk');
    localStorage.setItem('user', JSON.stringify(sampleUser));
    const { result } = renderHook(() => useAuth(), { wrapper });

    act(() => {
      result.current.updateUser({ name: 'Alice2', mustChangePW: true });
    });

    expect(result.current.user?.name).toBe('Alice2');
    expect(result.current.user?.mustChangePW).toBe(true);
    // Untouched fields preserved
    expect(result.current.user?.email).toBe('admin@a.com');
    const stored = JSON.parse(localStorage.getItem('user') ?? '{}');
    expect(stored.name).toBe('Alice2');
  });

  it('updateUser() is a no-op when no user is logged in', () => {
    const { result } = renderHook(() => useAuth(), { wrapper });

    act(() => {
      result.current.updateUser({ name: 'ghost' });
    });

    expect(result.current.user).toBeNull();
  });

  it('handles corrupted user JSON gracefully', () => {
    localStorage.setItem('user', 'not-json-at-all');
    const { result } = renderHook(() => useAuth(), { wrapper });
    expect(result.current.user).toBeNull();
  });

  it('isTranslator reflects role correctly', () => {
    localStorage.setItem('user', JSON.stringify({ ...sampleUser, role: 'translator' }));
    const { result } = renderHook(() => useAuth(), { wrapper });
    expect(result.current.isTranslator).toBe(true);
    expect(result.current.isAdmin).toBe(false);
  });
});
