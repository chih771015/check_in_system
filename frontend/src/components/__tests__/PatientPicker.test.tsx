import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import PatientPicker from '../PatientPicker';
import type { Patient } from '../../types';

const getPatientsMock = vi.fn();
vi.mock('../../api/patients', () => ({
  getPatients: (params: unknown) => getPatientsMock(params),
}));

const samplePatients: Patient[] = [
  { id: 1, name: 'Alice Wang', phone: '0900111222', idType: 'passport', idNumber: 'P001',
    createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' },
  { id: 2, name: 'Bob Chen', phone: '0900333444', idType: 'hn', idNumber: 'H001',
    createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' },
];

describe('PatientPicker', () => {
  beforeEach(() => {
    getPatientsMock.mockReset();
    document.body.innerHTML = '';
  });
  afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
  });

  it('fetches patients on mount', async () => {
    getPatientsMock.mockResolvedValueOnce({ data: samplePatients, total: 2, page: 1, pageSize: 20 });
    render(<PatientPicker value={undefined} onChange={() => {}} />);
    await vi.waitFor(() => {
      expect(getPatientsMock).toHaveBeenCalled();
    });
  });

  it('lists fetched patients as options', async () => {
    getPatientsMock.mockResolvedValueOnce({ data: samplePatients, total: 2, page: 1, pageSize: 20 });
    render(<PatientPicker value={undefined} onChange={() => {}} />);
    const user = userEvent.setup({ delay: null });

    await user.click(screen.getByRole('combobox'));

    // antd Select renders options into a portal under document.body
    expect(await screen.findByText(/Alice Wang/)).toBeInTheDocument();
    expect(await screen.findByText(/Bob Chen/)).toBeInTheDocument();
  });

  it('calls onChange when a patient is picked', async () => {
    getPatientsMock.mockResolvedValueOnce({ data: samplePatients, total: 2, page: 1, pageSize: 20 });
    const onChange = vi.fn();
    render(<PatientPicker value={undefined} onChange={onChange} />);
    const user = userEvent.setup({ delay: null });

    await user.click(screen.getByRole('combobox'));
    const opt = await screen.findByText(/Alice Wang/);
    await user.click(opt);

    // antd Select.onChange passes (value, optionObj); we only care about value.
    expect(onChange).toHaveBeenCalled();
    expect(onChange.mock.calls[0][0]).toBe(1);
  });

  it('debounces search input and re-queries', async () => {
    getPatientsMock
      .mockResolvedValueOnce({ data: samplePatients, total: 2, page: 1, pageSize: 20 })
      .mockResolvedValueOnce({ data: [samplePatients[0]], total: 1, page: 1, pageSize: 20 });
    render(<PatientPicker value={undefined} onChange={() => {}} />);
    const user = userEvent.setup({ delay: null });

    const combobox = screen.getByRole('combobox');
    await user.click(combobox);
    fireEvent.change(combobox, { target: { value: 'Alice' } });

    await vi.waitFor(() => {
      expect(getPatientsMock).toHaveBeenCalledTimes(2);
      // 第二次 call 應該帶 search=Alice
      const lastCall = getPatientsMock.mock.calls[1][0] as { search?: string };
      expect(lastCall.search).toBe('Alice');
    }, { timeout: 1000 });
  });

  it('shows selected patient name when value is set after mount', async () => {
    getPatientsMock.mockResolvedValueOnce({ data: samplePatients, total: 2, page: 1, pageSize: 20 });
    const { rerender } = render(<PatientPicker value={undefined} onChange={() => {}} />);
    await vi.waitFor(() => expect(getPatientsMock).toHaveBeenCalled());

    rerender(<PatientPicker value={1} onChange={() => {}} />);
    // selected label should appear in the combobox display
    expect(await screen.findByText(/Alice Wang/)).toBeInTheDocument();
  });
});
