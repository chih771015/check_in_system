import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup, waitFor, fireEvent } from '@testing-library/react';
import { App as AntApp } from 'antd';
import ScheduleManagement from '../ScheduleManagement';
import i18n from '../../../i18n';

const getAdminSchedulesMock = vi.fn();

vi.mock('../../../api/schedules', () => ({
  getAdminSchedules: (params: unknown) => getAdminSchedulesMock(params),
  createSchedule: vi.fn(),
  updateSchedule: vi.fn(),
  deleteSchedule: vi.fn(),
  deleteScheduleGroup: vi.fn(),
  importSchedules: vi.fn(),
}));
vi.mock('../../../api/translators', () => ({
  getTranslators: () => Promise.resolve([]),
}));
vi.mock('../../../api/checkins', () => ({
  adminUploadDiagnosis: vi.fn(),
  adminMarkNoShow: vi.fn(),
  adminSetActualAmount: vi.fn(),
  adminListDiagnosisPhotos: vi.fn(),
  adminDeleteDiagnosisPhoto: vi.fn(),
}));
vi.mock('../../../api/diagnosisResults', () => ({
  getSchedulePatientPhotos: vi.fn(),
}));
// Light stubs for heavy children so the test focuses on the filter state machine.
vi.mock('../../../components/SchedulePatientListEditor', () => ({ default: () => null }));
// Stub exposes a button that fires onUploaded, so we can drive the post-upload
// detail-modal refresh path without the real upload UI.
vi.mock('../../../components/DiagnosisUploadModal', () => ({
  default: ({ onUploaded }: { onUploaded: () => void }) => (
    <button onClick={() => onUploaded()}>trigger-uploaded</button>
  ),
}));
vi.mock('../../../components/NoShowModal', () => ({ default: () => null }));

function latestButton() {
  return screen.getByRole('button', { name: /Latest created/ });
}

// Apply a location filter via fireEvent (single render pass, far cheaper than
// userEvent typing on this heavy page).
function applyLocationFilter(value: string) {
  const input = screen.getByPlaceholderText('Search location');
  fireEvent.change(input, { target: { value } });
  fireEvent.keyDown(input, { key: 'Enter', code: 'Enter', charCode: 13 });
}

describe('ScheduleManagement — default sort + latest-created button', () => {
  beforeEach(async () => {
    getAdminSchedulesMock.mockReset();
    getAdminSchedulesMock.mockResolvedValue({ data: [], total: 0, page: 1, pageSize: 10 });
    document.body.innerHTML = '';
    await i18n.changeLanguage('en');
  });
  afterEach(() => { cleanup(); document.body.innerHTML = ''; });

  it('starts in default mode: fetches with no filters and highlights the button', async () => {
    render(<AntApp><ScheduleManagement /></AntApp>);
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenCalled());
    // No filters applied — only the pagination params travel with the request.
    expect(getAdminSchedulesMock).toHaveBeenLastCalledWith({ page: 1, pageSize: 10 });
    expect(latestButton()).toHaveClass('ant-btn-primary');
  });

  it('applying a filter un-highlights the button and queries with the filter', async () => {
    render(<AntApp><ScheduleManagement /></AntApp>);
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenCalled());

    applyLocationFilter('VGH');

    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenLastCalledWith({ location: 'VGH', page: 1, pageSize: 10 }));
    expect(latestButton()).not.toHaveClass('ant-btn-primary');
  });

  it('pressing the button clears filters and returns to highlighted default mode', async () => {
    render(<AntApp><ScheduleManagement /></AntApp>);
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenCalled());

    applyLocationFilter('VGH');
    await waitFor(() => expect(latestButton()).not.toHaveClass('ant-btn-primary'));

    fireEvent.click(latestButton());
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenLastCalledWith({ page: 1, pageSize: 10 }));
    expect(latestButton()).toHaveClass('ant-btn-primary');
  });

  // Regression: opening the edit modal must preload the date + start/end time
  // pickers from the existing schedule. Previously openEdit only set
  // translator/location/note, so the time fields rendered empty.
  it('preloads date and start/end time into the edit form', async () => {
    const sched = {
      id: 42, date: '2026-01-15', startTime: '09:00', endTime: '12:00',
      location: 'VGH', translatorId: 1, translatorName: 'T', status: 'pending',
      patients: [],
    };
    getAdminSchedulesMock.mockResolvedValue({ data: [sched], total: 1, page: 1, pageSize: 10 });
    render(<AntApp><ScheduleManagement /></AntApp>);
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenCalled());

    fireEvent.click(screen.getByRole('button', { name: /Edit/ }));

    await waitFor(() => expect(screen.getByDisplayValue('2026-01-15')).toBeInTheDocument());
    expect(screen.getByDisplayValue('09:00')).toBeInTheDocument();
    expect(screen.getByDisplayValue('12:00')).toBeInTheDocument();
  });

  // Regression: after the default view was capped to recent-created rows, the
  // detail-modal refresh must re-fetch with the ACTIVE filter (so the open
  // record is present), never with {} — which could miss an older schedule.
  it('refreshes the open detail modal using the active filter, not an unfiltered fetch', async () => {
    const sched = {
      id: 42, date: '2026-01-15', startTime: '09:00', endTime: '12:00',
      location: 'VGH', translatorId: 1, translatorName: 'T', status: 'pending',
      patients: [{
        id: 7, patientId: 3, patientName: 'P', patientPhone: '0900', idType: 'passport',
        idNumber: 'X1', status: 'pending', startTime: '09:00', endTime: '10:00',
        prepaidAmount: 0, actualAmount: 0,
      }],
    };
    getAdminSchedulesMock.mockResolvedValue({ data: [sched], total: 1, page: 1, pageSize: 10 });
    render(<AntApp><ScheduleManagement /></AntApp>);
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenCalled());

    applyLocationFilter('VGH');
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenLastCalledWith({ location: 'VGH', page: 1, pageSize: 10 }));

    fireEvent.click(screen.getByRole('button', { name: /Detail/ }));
    fireEvent.click(await screen.findByRole('button', { name: /Upload/ }));

    getAdminSchedulesMock.mockClear();
    fireEvent.click(screen.getByText('trigger-uploaded'));

    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenCalledWith({ location: 'VGH', page: 1, pageSize: 10 }));
    expect(getAdminSchedulesMock).not.toHaveBeenCalledWith({ page: 1, pageSize: 10 });
  });
});
