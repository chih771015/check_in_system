import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup, waitFor, fireEvent } from '@testing-library/react';
import { MemoryRouter, Routes, Route, useNavigate } from 'react-router-dom';
import { App as AntApp } from 'antd';
import AppLayout from '../AppLayout';
import i18n from '../../i18n';

const getMonthlyTotalMock = vi.fn();
const useAuthMock = vi.fn();

vi.mock('../../api/stats', () => ({
  getMonthlyTotal: () => getMonthlyTotalMock(),
}));
vi.mock('../../stores/authStore', () => ({
  useAuth: () => useAuthMock(),
}));

const baseAuth = {
  user: { id: 1, name: 'Admin', role: 'admin' },
  login: vi.fn(),
  logout: vi.fn(),
};

function renderLayout() {
  return render(
    <AntApp>
      <MemoryRouter>
        <AppLayout />
      </MemoryRouter>
    </AntApp>,
  );
}

describe('AppLayout — current-month expenditure banner', () => {
  beforeEach(async () => {
    getMonthlyTotalMock.mockReset();
    useAuthMock.mockReset();
    getMonthlyTotalMock.mockResolvedValue({ yearMonth: '2026-06', total: 12345 });
    document.body.innerHTML = '';
    await i18n.changeLanguage('en');
  });
  afterEach(() => { cleanup(); document.body.innerHTML = ''; });

  it('shows the monthly total banner for admins', async () => {
    useAuthMock.mockReturnValue({ ...baseAuth, isAdmin: true });
    renderLayout();
    await waitFor(() => expect(getMonthlyTotalMock).toHaveBeenCalled());
    expect(await screen.findByText(/NT\$ 12,345/)).toBeInTheDocument();
    expect(screen.getByText(/2026-06/)).toBeInTheDocument();
  });

  it('re-fetches the banner total on navigation so it does not go stale', async () => {
    useAuthMock.mockReturnValue({ ...baseAuth, isAdmin: true });
    const Go = () => {
      const navigate = useNavigate();
      return <button onClick={() => navigate('/admin/schedules')}>go</button>;
    };
    render(
      <AntApp>
        <MemoryRouter initialEntries={['/admin/patients']}>
          <Routes>
            <Route path="/admin" element={<AppLayout />}>
              <Route path="patients" element={<Go />} />
              <Route path="schedules" element={<div>sched</div>} />
            </Route>
          </Routes>
        </MemoryRouter>
      </AntApp>,
    );
    await waitFor(() => expect(getMonthlyTotalMock).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByText('go'));
    // Navigating to another admin route must trigger a fresh fetch.
    await waitFor(() => expect(getMonthlyTotalMock).toHaveBeenCalledTimes(2));
  });

  it('does not fetch or show the banner for non-admins', async () => {
    useAuthMock.mockReturnValue({ ...baseAuth, user: { id: 2, name: 'T', role: 'translator' }, isAdmin: false });
    renderLayout();
    // Give any stray effect a tick; the banner must never appear.
    await Promise.resolve();
    expect(getMonthlyTotalMock).not.toHaveBeenCalled();
    expect(screen.queryByText(/NT\$/)).toBeNull();
  });
});
