import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { App as AntApp } from 'antd';
import PatientHistory from '../PatientHistory';
import i18n from '../../../i18n';

const getPatientHistoryMock = vi.fn();

vi.mock('../../../api/patients', () => ({
  getPatientHistory: (id: number, range?: unknown) => getPatientHistoryMock(id, range),
}));

const patient = {
  id: 7, name: 'P', phone: '1', idType: 'passport' as const, idNumber: 'X1',
  createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z',
};
const entry = {
  scheduleId: 1, date: '2026-05-10', startTime: '09:00', endTime: '10:00',
  location: 'L', translatorName: 'Alice', status: 'completed', diagnosisPhotos: [],
  prepaidAmount: 0, actualAmount: 300,
};

function renderPage() {
  return render(
    <AntApp>
      <MemoryRouter initialEntries={['/admin/patients/7/history']}>
        <Routes>
          <Route path="/admin/patients/:id/history" element={<PatientHistory />} />
        </Routes>
      </MemoryRouter>
    </AntApp>,
  );
}

describe('PatientHistory — date range + actual total', () => {
  beforeEach(async () => {
    getPatientHistoryMock.mockReset();
    getPatientHistoryMock.mockResolvedValue({ patient, history: [entry], actualTotal: 300 });
    document.body.innerHTML = '';
    await i18n.changeLanguage('en');
  });
  afterEach(() => { cleanup(); document.body.innerHTML = ''; });

  it('fetches with no range on mount and shows the actual total', async () => {
    renderPage();
    await waitFor(() => expect(getPatientHistoryMock).toHaveBeenCalled());
    expect(getPatientHistoryMock).toHaveBeenLastCalledWith(7, { dateFrom: undefined, dateTo: undefined });
    expect(await screen.findByText('Actual paid total')).toBeInTheDocument();
    expect(screen.getByText('300')).toBeInTheDocument();
  });

  it('renders the date-range filter for narrowing the history', async () => {
    renderPage();
    await waitFor(() => expect(getPatientHistoryMock).toHaveBeenCalled());
    // Both ends of the RangePicker are present so the user can filter by date.
    expect(document.querySelector('input[placeholder="Start date"]')).toBeInTheDocument();
    expect(document.querySelector('input[placeholder="End date"]')).toBeInTheDocument();
  });
});
