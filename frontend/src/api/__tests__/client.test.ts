import { describe, it, expect, beforeEach, vi } from 'vitest';
import type { AxiosError } from 'axios';
import { mapErrorResponse, unwrapResponse } from '../client';
import i18n from '../../i18n';

function makeError(status: number, data?: { code?: string; message?: string }): AxiosError<{ code?: string; message?: string }> {
  return {
    isAxiosError: true,
    name: 'AxiosError',
    message: data?.message ?? '',
    response: {
      status,
      data: data ?? {},
      statusText: '',
      headers: {},
      config: {} as never,
    },
    config: {} as never,
    toJSON: () => ({}),
  } as AxiosError<{ code?: string; message?: string }>;
}

describe('unwrapResponse', () => {
  it('strips single-key { data: ... } envelope', () => {
    expect(unwrapResponse({ data: [1, 2, 3] })).toEqual([1, 2, 3]);
  });

  it('returns body untouched when envelope has more keys', () => {
    const body = { data: [], total: 5 };
    expect(unwrapResponse(body)).toEqual(body);
  });

  it('returns null / primitives untouched', () => {
    expect(unwrapResponse(null)).toBeNull();
    expect(unwrapResponse('hello')).toBe('hello');
    expect(unwrapResponse(42)).toBe(42);
  });
});

describe('mapErrorResponse', () => {
  beforeEach(async () => {
    localStorage.clear();
    await i18n.changeLanguage('en');
  });

  it('translates known error code into i18n message (en)', () => {
    const err = makeError(401, { code: 'INVALID_CREDENTIALS', message: 'raw backend msg' });
    const mapped = mapErrorResponse(err, () => {});
    expect(mapped.translatedMessage).toBe('Invalid email or password');
    expect(err.response!.data.message).toBe('Invalid email or password');
  });

  it('translates the same code into Chinese after switching language', async () => {
    await i18n.changeLanguage('zh-TW');
    const err = makeError(401, { code: 'INVALID_CREDENTIALS', message: 'raw' });
    const mapped = mapErrorResponse(err, () => {});
    expect(mapped.translatedMessage).toBe('帳號或密碼錯誤');
  });

  it('falls back to backend message when code is unknown', () => {
    const err = makeError(500, { code: 'TOTALLY_NEW_CODE_NEVER_SEEN', message: 'backend fallback' });
    const mapped = mapErrorResponse(err, () => {});
    expect(mapped.translatedMessage).toBe('backend fallback');
  });

  it('redirects to /login and clears auth on 401', () => {
    localStorage.setItem('token', 'abc');
    localStorage.setItem('user', '{}');
    const redirect = vi.fn();

    const err = makeError(401, { code: 'INVALID_CREDENTIALS' });
    mapErrorResponse(err, redirect);

    expect(redirect).toHaveBeenCalledWith('/login');
    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('user')).toBeNull();
  });

  it('redirects to /change-password on 403 PASSWORD_CHANGE_REQUIRED', () => {
    const redirect = vi.fn();
    const err = makeError(403, { code: 'PASSWORD_CHANGE_REQUIRED' });
    mapErrorResponse(err, redirect, () => '/admin/translators');
    expect(redirect).toHaveBeenCalledWith('/change-password');
  });

  it('does NOT redirect again when already on /change-password', () => {
    const redirect = vi.fn();
    const err = makeError(403, { code: 'PASSWORD_CHANGE_REQUIRED' });
    mapErrorResponse(err, redirect, () => '/change-password');
    expect(redirect).not.toHaveBeenCalled();
  });

  it('does not crash when response payload is missing', () => {
    const err = {
      isAxiosError: true,
      name: 'AxiosError',
      message: 'network error',
      config: {} as never,
      toJSON: () => ({}),
    } as AxiosError<{ code?: string; message?: string }>;
    expect(() => mapErrorResponse(err, () => {})).not.toThrow();
  });
});
