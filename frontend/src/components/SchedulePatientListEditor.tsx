import { useEffect } from 'react';
import { Button, Space, TimePicker, Card } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import PatientPicker from './PatientPicker';
import type { SchedulePatientPayload } from '../types';
import { clampPatientTimes } from '../utils/schedulePatient';

interface SchedulePatientListEditorProps {
  /** Current list of patient slots. */
  value: SchedulePatientPayload[];
  /** Called whenever the list changes (add/remove/edit a row). */
  onChange: (value: SchedulePatientPayload[]) => void;
  /** Overall schedule start time, used for default values of new rows. */
  overallStart: string;
  /** Overall schedule end time, used for default values of new rows. */
  overallEnd: string;
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
    ? [{ patientId: 0, startTime: overallStart, endTime: overallEnd }]
    : value;

  const updateRow = (idx: number, patch: Partial<SchedulePatientPayload>) => {
    const next = rows.map((r, i) => (i === idx ? { ...r, ...patch } : r));
    onChange(next);
  };
  const removeRow = (idx: number) => {
    onChange(rows.filter((_, i) => i !== idx));
  };
  const addRow = () => {
    onChange([...rows, { patientId: 0, startTime: overallStart, endTime: overallEnd }]);
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
