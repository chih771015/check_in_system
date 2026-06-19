import { useEffect, useState } from 'react';
import { Button, Space, TimePicker, Card, InputNumber, Typography } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import PatientPicker from './PatientPicker';
import type { SchedulePatientPayload } from '../types';
import { clampPatientTimes } from '../utils/schedulePatient';
import { getPatientActualTotal } from '../api/patients';

/**
 * PatientYearPaid shows, read-only, how much a patient has already been paid
 * (actual_amount) during the schedule's year. Re-fetches when the patient or
 * year changes. Renders nothing until a patient is selected.
 */
function PatientYearPaid({ patientId, year }: { patientId?: number; year?: number }) {
  const { t } = useTranslation();
  const [total, setTotal] = useState<number | null>(null);

  useEffect(() => {
    if (!patientId || !year) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- clear stale total when patient/year cleared
      setTotal(null);
      return;
    }
    let active = true;
    getPatientActualTotal(patientId, year)
      .then((r) => { if (active) setTotal(r.total); })
      .catch(() => { if (active) setTotal(null); });
    return () => { active = false; };
  }, [patientId, year]);

  if (!patientId || !year || total === null) return null;
  return (
    <Typography.Text type="secondary">
      {t('schedules.yearActualPaid', { year })}: NT$ {total.toLocaleString()}
    </Typography.Text>
  );
}

interface SchedulePatientListEditorProps {
  /** Current list of patient slots. */
  value: SchedulePatientPayload[];
  /** Called whenever the list changes (add/remove/edit a row). */
  onChange: (value: SchedulePatientPayload[]) => void;
  /** Overall schedule start time, used for default values of new rows. */
  overallStart: string;
  /** Overall schedule end time, used for default values of new rows. */
  overallEnd: string;
  /** Year of the schedule being edited; drives the per-patient已實付 hint. */
  scheduleYear?: number;
  /** Disables all controls (e.g. while parent form is submitting). */
  disabled?: boolean;
}

/**
 * SchedulePatientListEditor lets an admin add/remove/edit patient slots
 * inside the schedule create/edit modal. Each row pairs a PatientPicker with
 * a start/end TimePicker.
 *
 * If `value` is empty on first render we show one blank row so the user
 * always has somewhere to start — they never see a "no patients" placeholder.
 */
export default function SchedulePatientListEditor({
  value,
  onChange,
  overallStart,
  overallEnd,
  scheduleYear,
  disabled,
}: SchedulePatientListEditorProps) {
  const { t } = useTranslation();

  // When overall start/end changes, clamp existing rows so they stay inside
  // the new range. clampPatientTimes returns the same array reference when
  // nothing changed, so this effect is no-op for already-valid edits.
  useEffect(() => {
    const clamped = clampPatientTimes(overallStart, overallEnd, value);
    if (clamped !== value) onChange(clamped);
    // We intentionally exclude `value` / `onChange` from deps — we only want
    // to react to overall range changes, not to typing inside the editor.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [overallStart, overallEnd]);

  // Display value: empty list shown as one blank row so the UI is never empty.
  const rows: SchedulePatientPayload[] = value.length === 0
    ? [{ patientId: 0, startTime: overallStart, endTime: overallEnd, prepaidAmount: 0 }]
    : value;

  const updateRow = (idx: number, patch: Partial<SchedulePatientPayload>) => {
    const next = rows.map((r, i) => (i === idx ? { ...r, ...patch } : r));
    onChange(next);
  };
  const removeRow = (idx: number) => {
    onChange(rows.filter((_, i) => i !== idx));
  };
  const addRow = () => {
    onChange([...rows, { patientId: 0, startTime: overallStart, endTime: overallEnd, prepaidAmount: 0 }]);
  };

  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      {rows.map((row, idx) => (
        <Card key={idx} size="small" bodyStyle={{ padding: 12 }}>
          <Space direction="vertical" style={{ width: '100%' }} size="small">
            <PatientPicker
              value={row.patientId || undefined}
              onChange={(pid) => updateRow(idx, { patientId: pid })}
              disabled={disabled}
            />
            <PatientYearPaid patientId={row.patientId || undefined} year={scheduleYear} />
            <Space>
              <TimePicker
                format="HH:mm"
                value={row.startTime ? dayjs(row.startTime, 'HH:mm') : null}
                onChange={(d) => updateRow(idx, { startTime: d ? d.format('HH:mm') : '' })}
                disabled={disabled}
                placeholder={t('schedules.startTime')}
              />
              <TimePicker
                format="HH:mm"
                value={row.endTime ? dayjs(row.endTime, 'HH:mm') : null}
                onChange={(d) => updateRow(idx, { endTime: d ? d.format('HH:mm') : '' })}
                disabled={disabled}
                placeholder={t('schedules.endTime')}
              />
              <Button
                icon={<DeleteOutlined />}
                onClick={() => removeRow(idx)}
                disabled={disabled}
                danger
                aria-label="Delete"
              >
                {t('common.delete')}
              </Button>
            </Space>
            <Space>
              <Typography.Text>{t('schedules.prepaidAmount')}</Typography.Text>
              <InputNumber
                min={0}
                precision={0}
                value={row.prepaidAmount}
                onChange={(v) => updateRow(idx, { prepaidAmount: typeof v === 'number' ? v : 0 })}
                disabled={disabled}
                style={{ width: 140 }}
                aria-label={t('schedules.prepaidAmount')}
              />
            </Space>
          </Space>
        </Card>
      ))}
      <Button
        type="dashed"
        icon={<PlusOutlined />}
        onClick={addRow}
        disabled={disabled}
        block
        aria-label="Add"
      >
        {t('common.add')}
      </Button>
    </Space>
  );
}
