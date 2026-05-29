import type { SchedulePatientPayload } from '../types';

/**
 * clampPatientTimes pulls each row's start/end inside [overallStart, overallEnd].
 *
 * Returns the SAME array reference when nothing changed so callers (notably
 * the useEffect inside SchedulePatientListEditor) don't retrigger themselves.
 *
 * Rows whose startTime / endTime is empty (a freshly-inserted blank row) are
 * left alone — they aren't part of the user's data yet.
 */
export function clampPatientTimes(
  overallStart: string,
  overallEnd: string,
  rows: SchedulePatientPayload[],
): SchedulePatientPayload[] {
  let changed = false;
  const next = rows.map((row) => {
    if (!row.startTime || !row.endTime) return row;
    let startTime = row.startTime;
    let endTime = row.endTime;
    if (startTime < overallStart) {
      startTime = overallStart;
    }
    if (endTime > overallEnd) {
      endTime = overallEnd;
    }
    // After clamping both ends, if start > end we collapse end to start to
    // keep the structural invariant; validate() then flags this row.
    if (startTime > endTime) {
      endTime = startTime;
    }
    if (startTime === row.startTime && endTime === row.endTime) {
      return row;
    }
    changed = true;
    return { ...row, startTime, endTime };
  });
  return changed ? next : rows;
}

export type ValidationResult =
  | { ok: true }
  | { ok: false; code: 'SCHEDULE_PATIENTS_REQUIRED' | 'PATIENT_END_BEFORE_START' | 'PATIENT_TIME_OUT_OF_RANGE' | 'DUPLICATE_PATIENT_IN_SCHEDULE' };

/**
 * validatePatientTimes mirrors the backend rules in ScheduleService so the UI
 * can fail-fast with a translated message instead of round-tripping to the
 * server. The error codes returned match dto/error.go constants so callers
 * can reuse the same i18n keys (`errors.<CODE>`).
 *
 * Order of checks matches the backend for parity:
 *   1. at least one patient with a real patientId
 *   2. each row's end > start
 *   3. each row inside [overallStart, overallEnd]
 *   4. no duplicate patientId within the schedule
 */
export function validatePatientTimes(
  overallStart: string,
  overallEnd: string,
  rows: SchedulePatientPayload[],
): ValidationResult {
  const filled = rows.filter((r) => r.patientId && r.startTime && r.endTime);
  if (filled.length === 0) {
    return { ok: false, code: 'SCHEDULE_PATIENTS_REQUIRED' };
  }
  for (const r of filled) {
    if (r.endTime <= r.startTime) {
      return { ok: false, code: 'PATIENT_END_BEFORE_START' };
    }
    if (r.startTime < overallStart || r.endTime > overallEnd) {
      return { ok: false, code: 'PATIENT_TIME_OUT_OF_RANGE' };
    }
  }
  const seen = new Set<number>();
  for (const r of filled) {
    if (seen.has(r.patientId)) {
      return { ok: false, code: 'DUPLICATE_PATIENT_IN_SCHEDULE' };
    }
    seen.add(r.patientId);
  }
  return { ok: true };
}
