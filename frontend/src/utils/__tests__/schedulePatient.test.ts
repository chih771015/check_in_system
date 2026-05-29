import { describe, it, expect } from 'vitest';
import { clampPatientTimes, validatePatientTimes } from '../schedulePatient';
import type { SchedulePatientPayload } from '../../types';

describe('clampPatientTimes', () => {
  it('clamps startTime below overallStart up to overallStart', () => {
    const got = clampPatientTimes('09:00', '12:00', [
      { patientId: 1, startTime: '08:00', endTime: '10:00' },
    ]);
    expect(got[0].startTime).toBe('09:00');
    expect(got[0].endTime).toBe('10:00');
  });

  it('clamps endTime above overallEnd down to overallEnd', () => {
    const got = clampPatientTimes('09:00', '12:00', [
      { patientId: 1, startTime: '10:00', endTime: '13:00' },
    ]);
    expect(got[0].endTime).toBe('12:00');
  });

  it('leaves rows entirely outside the new range still inside it', () => {
    // patient 06:00-07:00 with overall 09:00-12:00 → both ends collapse to 09:00.
    // Caller should detect this degenerate case via validate; clamp must not crash.
    const got = clampPatientTimes('09:00', '12:00', [
      { patientId: 1, startTime: '06:00', endTime: '07:00' },
    ]);
    expect(got[0].startTime).toBe('09:00');
    expect(got[0].endTime).toBe('09:00');
  });

  it('leaves rows already in range untouched', () => {
    const input: SchedulePatientPayload[] = [
      { patientId: 1, startTime: '09:30', endTime: '10:30' },
    ];
    const got = clampPatientTimes('09:00', '12:00', input);
    expect(got[0]).toEqual(input[0]);
  });

  it('returns same reference when nothing changes (so React skips re-render)', () => {
    const input: SchedulePatientPayload[] = [
      { patientId: 1, startTime: '09:30', endTime: '10:30' },
    ];
    const got = clampPatientTimes('09:00', '12:00', input);
    // Same array reference avoids retriggering useEffect.
    expect(got).toBe(input);
  });

  it('skips rows with empty startTime / endTime', () => {
    const input: SchedulePatientPayload[] = [
      { patientId: 0, startTime: '', endTime: '' },
    ];
    expect(clampPatientTimes('09:00', '12:00', input)).toBe(input);
  });
});

describe('validatePatientTimes', () => {
  it('returns OK_EMPTY when no patients selected', () => {
    expect(validatePatientTimes('09:00', '12:00', [])).toEqual({
      ok: false,
      code: 'SCHEDULE_PATIENTS_REQUIRED',
    });
  });

  it('returns OK when all rows are within range and ordered', () => {
    expect(
      validatePatientTimes('09:00', '12:00', [
        { patientId: 1, startTime: '09:00', endTime: '10:00' },
        { patientId: 2, startTime: '10:00', endTime: '11:00' },
      ]),
    ).toEqual({ ok: true });
  });

  it('detects end <= start', () => {
    expect(
      validatePatientTimes('09:00', '12:00', [
        { patientId: 1, startTime: '10:00', endTime: '10:00' },
      ]),
    ).toEqual({ ok: false, code: 'PATIENT_END_BEFORE_START' });
  });

  it('detects start < overallStart', () => {
    expect(
      validatePatientTimes('09:00', '12:00', [
        { patientId: 1, startTime: '08:00', endTime: '10:00' },
      ]),
    ).toEqual({ ok: false, code: 'PATIENT_TIME_OUT_OF_RANGE' });
  });

  it('detects end > overallEnd', () => {
    expect(
      validatePatientTimes('09:00', '12:00', [
        { patientId: 1, startTime: '11:00', endTime: '13:00' },
      ]),
    ).toEqual({ ok: false, code: 'PATIENT_TIME_OUT_OF_RANGE' });
  });

  it('detects duplicate patient ids', () => {
    expect(
      validatePatientTimes('09:00', '12:00', [
        { patientId: 1, startTime: '09:00', endTime: '10:00' },
        { patientId: 1, startTime: '10:00', endTime: '11:00' },
      ]),
    ).toEqual({ ok: false, code: 'DUPLICATE_PATIENT_IN_SCHEDULE' });
  });

  it('detects missing patientId', () => {
    expect(
      validatePatientTimes('09:00', '12:00', [
        { patientId: 0, startTime: '09:00', endTime: '10:00' },
      ]),
    ).toEqual({ ok: false, code: 'SCHEDULE_PATIENTS_REQUIRED' });
  });
});
