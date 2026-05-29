import { describe, it, expect } from 'vitest';
import { extractApiError } from '../apiError';

describe('extractApiError', () => {
  it('prefers translatedMessage when set by axios interceptor', () => {
    const err = {
      translatedMessage: 'Patient time slot is outside the schedule range',
      response: { data: { message: 'patient time slot is outside ...', code: 'PATIENT_TIME_OUT_OF_RANGE' } },
    };
    expect(extractApiError(err)).toBe('Patient time slot is outside the schedule range');
  });

  it('falls back to response.data.message when no translatedMessage', () => {
    const err = { response: { data: { message: 'raw backend message' } } };
    expect(extractApiError(err)).toBe('raw backend message');
  });

  it('returns undefined when nothing useful is present (e.g. network error)', () => {
    expect(extractApiError({})).toBeUndefined();
    expect(extractApiError(null)).toBeUndefined();
    expect(extractApiError(undefined)).toBeUndefined();
  });

  it('handles 413 Payload Too Large (no payload from server)', () => {
    // nginx/gin may reject before a JSON body is produced; only the
    // request meta is in the axios error. We surface a hint instead of
    // generic "Failed" so users know the file is too big.
    const err = { response: { status: 413, statusText: 'Request Entity Too Large', data: '' } };
    expect(extractApiError(err)).toBe('File too large');
  });

  it('handles plain network error (no response at all)', () => {
    const err = { message: 'Network Error' } as unknown;
    expect(extractApiError(err)).toBe('Network Error');
  });
});
