import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import SchedulePatientListEditor from '../SchedulePatientListEditor';
import type { SchedulePatientPayload } from '../../types';

const getPatientActualTotalMock = vi.fn();

// Mock the patient API so PatientPicker (inside the editor) doesn't fetch.
vi.mock('../../api/patients', () => ({
  getPatients: vi.fn().mockResolvedValue({ data: [], total: 0, page: 1, pageSize: 20 }),
  getPatientActualTotal: (id: number, year: number) => getPatientActualTotalMock(id, year),
}));

function Harness({ initial, scheduleYear }: { initial: SchedulePatientPayload[]; scheduleYear?: number }) {
  const [value, setValue] = useState(initial);
  return (
    <SchedulePatientListEditor
      value={value}
      onChange={setValue}
      overallStart="09:00"
      overallEnd="12:00"
      scheduleYear={scheduleYear}
    />
  );
}

describe('SchedulePatientListEditor', () => {
  beforeEach(() => {
    getPatientActualTotalMock.mockReset();
    getPatientActualTotalMock.mockResolvedValue({ year: 2026, total: 8000 });
    document.body.innerHTML = '';
  });
  afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
  });

  it('renders one empty row when initial value is empty', () => {
    render(<Harness initial={[]} />);
    // Should always show at least one slot ready to fill.
    expect(screen.getAllByRole('combobox').length).toBeGreaterThanOrEqual(1);
  });

  it('renders existing rows from the value prop', () => {
    render(
      <Harness
        initial={[
          { patientId: 1, startTime: '09:00', endTime: '10:00', prepaidAmount: 0 },
          { patientId: 2, startTime: '10:00', endTime: '11:00', prepaidAmount: 0 },
        ]}
      />,
    );
    // Each row gets a delete button — 2 rows ⇒ 2 delete buttons.
    expect(screen.getAllByRole('button', { name: /delete|remove|刪除/i }).length).toBeGreaterThanOrEqual(2);
  });

  it('clicking Add appends a new empty row', async () => {
    render(<Harness initial={[{ patientId: 1, startTime: '09:00', endTime: '10:00', prepaidAmount: 0 }]} />);
    const user = userEvent.setup({ delay: null });

    const before = screen.getAllByRole('combobox').length;
    await user.click(screen.getByRole('button', { name: /add|新增/i }));
    const after = screen.getAllByRole('combobox').length;
    expect(after).toBe(before + 1);
  });

  it('shows the patient年度已實付 hint when a patient and schedule year are set', async () => {
    render(<Harness initial={[{ patientId: 5, startTime: '09:00', endTime: '10:00', prepaidAmount: 0 }]} scheduleYear={2026} />);
    await waitFor(() => expect(getPatientActualTotalMock).toHaveBeenCalledWith(5, 2026));
    expect(await screen.findByText(/8,000/)).toBeInTheDocument();
  });

  it('does not query the year total when no schedule year is provided', () => {
    render(<Harness initial={[{ patientId: 5, startTime: '09:00', endTime: '10:00', prepaidAmount: 0 }]} />);
    expect(getPatientActualTotalMock).not.toHaveBeenCalled();
  });

  it('clicking Delete removes a row', async () => {
    render(
      <Harness
        initial={[
          { patientId: 1, startTime: '09:00', endTime: '10:00', prepaidAmount: 0 },
          { patientId: 2, startTime: '10:00', endTime: '11:00', prepaidAmount: 0 },
        ]}
      />,
    );
    const user = userEvent.setup({ delay: null });

    const deleteButtons = screen.getAllByRole('button', { name: /delete|remove|刪除/i });
    await user.click(deleteButtons[0]);

    // Only one combobox should remain after deletion.
    expect(screen.getAllByRole('combobox')).toHaveLength(1);
  });
});
