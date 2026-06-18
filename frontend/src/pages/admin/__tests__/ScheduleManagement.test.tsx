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
vi.mock('../../../components/DiagnosisUploadModal', () => ({ default: () => null }));
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
    getAdminSchedulesMock.mockResolvedValue([]);
    document.body.innerHTML = '';
    await i18n.changeLanguage('en');
  });
  afterEach(() => { cleanup(); document.body.innerHTML = ''; });

  it('starts in default mode: fetches with no filters and highlights the button', async () => {
    render(<AntApp><ScheduleManagement /></AntApp>);
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenCalled());
    expect(getAdminSchedulesMock).toHaveBeenLastCalledWith({});
    expect(latestButton()).toHaveClass('ant-btn-primary');
  });

  it('applying a filter un-highlights the button and queries with the filter', async () => {
    render(<AntApp><ScheduleManagement /></AntApp>);
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenCalled());

    applyLocationFilter('VGH');

    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenLastCalledWith({ location: 'VGH' }));
    expect(latestButton()).not.toHaveClass('ant-btn-primary');
  });

  it('pressing the button clears filters and returns to highlighted default mode', async () => {
    render(<AntApp><ScheduleManagement /></AntApp>);
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenCalled());

    applyLocationFilter('VGH');
    await waitFor(() => expect(latestButton()).not.toHaveClass('ant-btn-primary'));

    fireEvent.click(latestButton());
    await waitFor(() => expect(getAdminSchedulesMock).toHaveBeenLastCalledWith({}));
    expect(latestButton()).toHaveClass('ant-btn-primary');
  });
});
