import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, within, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App as AntApp } from 'antd';
import DiagnosisResultsPage from '../DiagnosisResults';
import i18n from '../../../i18n';

const getDiagnosisResultsMock = vi.fn();

vi.mock('../../../api/diagnosisResults', () => ({
  getDiagnosisResults: (q: unknown) => getDiagnosisResultsMock(q),
}));
vi.mock('../../../api/translators', () => ({
  getTranslators: () => Promise.resolve([]),
}));

// Stub the heavy modals so the test asserts wiring (which sp id, which modal).
vi.mock('../../../components/DiagnosisUploadModal', () => ({
  default: ({ open, schedulePatientId }: { open: boolean; schedulePatientId: number }) =>
    open ? <div data-testid="manage-modal">manage:{schedulePatientId}</div> : null,
}));
vi.mock('../../../components/NoShowModal', () => ({
  default: ({ open, schedulePatientId }: { open: boolean; schedulePatientId: number }) =>
    open ? <div data-testid="noshow-modal">noshow:{schedulePatientId}</div> : null,
}));

const completedRow = {
  schedulePatientId: 11, scheduleId: 1, date: '2026-06-10', startTime: '09:00', endTime: '10:00',
  location: 'L', note: '', translatorId: 1, translatorName: 'Alice', patientId: 1,
  patientName: 'PatientOne', patientPhone: '1', idType: 'passport' as const, idNumber: 'X',
  status: 'completed' as const, diagnosisPhotos: ['/u/a.jpg'], updatedAt: '2026-06-10T00:00:00Z',
};
const noShowRow = {
  ...completedRow, schedulePatientId: 22, patientName: 'PatientTwo',
  status: 'no_show' as const, noShowReason: 'absent', diagnosisPhotos: [],
};

function renderPage() {
  return render(<AntApp><DiagnosisResultsPage /></AntApp>);
}

describe('DiagnosisResults — admin edit from overview', () => {
  beforeEach(async () => {
    getDiagnosisResultsMock.mockReset();
    getDiagnosisResultsMock.mockResolvedValue({ data: [completedRow, noShowRow], total: 2, page: 1, pageSize: 20 });
    document.body.innerHTML = '';
    await i18n.changeLanguage('en');
  });
  afterEach(() => { cleanup(); document.body.innerHTML = ''; });

  it('completed row opens the manage-photos modal with its schedulePatientId', async () => {
    renderPage();
    const user = userEvent.setup({ delay: null });
    await screen.findByText('PatientOne');

    const row = screen.getByText('PatientOne').closest('tr')!;
    await user.click(within(row).getByRole('button', { name: /Manage Diagnosis Photos/ }));

    expect(await screen.findByTestId('manage-modal')).toHaveTextContent('manage:11');
  });

  it('completed row can mark no-show; no_show row cannot', async () => {
    renderPage();
    // (heavy page + Popconfirm + modal chain — give it room in CI)
    const user = userEvent.setup({ delay: null });
    await screen.findByText('PatientOne');

    const completed = screen.getByText('PatientOne').closest('tr')!;
    expect(within(completed).queryByRole('button', { name: /Mark No-Show/ })).toBeInTheDocument();

    const noShow = screen.getByText('PatientTwo').closest('tr')!;
    expect(within(noShow).queryByRole('button', { name: /Mark No-Show/ })).toBeNull();
    // but a no_show row can still be managed (e.g. upload to restore completed)
    expect(within(noShow).getByRole('button', { name: /Manage Diagnosis Photos/ })).toBeInTheDocument();

    // The no-show button is guarded by a Popconfirm (it purges photos).
    await user.click(within(completed).getByRole('button', { name: /Mark No-Show/ }));
    await user.click(await screen.findByRole('button', { name: 'Confirm' }));
    expect(await screen.findByTestId('noshow-modal')).toHaveTextContent('noshow:11');
  }, 30000);
});
