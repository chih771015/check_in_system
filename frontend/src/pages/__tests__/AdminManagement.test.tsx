import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, within, cleanup, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { App as AntApp } from 'antd';
import AdminManagement from '../admin/AdminManagement';
import { AuthProvider } from '../../stores/authStore';
import i18n from '../../i18n';

const getAdminsMock = vi.fn();
const createAdminMock = vi.fn();
const deleteAdminMock = vi.fn();

vi.mock('../../api/admins', () => ({
  getAdmins: () => getAdminsMock(),
  createAdmin: (data: unknown) => createAdminMock(data),
  deleteAdmin: (id: number) => deleteAdminMock(id),
}));

const sampleAdmins = [
  { id: 1, email: 'me@a.com', name: 'Me', status: 'active' as const, createdAt: '2026-01-01T00:00:00Z' },
  { id: 2, email: 'other@a.com', name: 'Other', status: 'active' as const, createdAt: '2026-02-01T00:00:00Z' },
];

function renderPage() {
  localStorage.setItem('token', 'tk');
  localStorage.setItem('user', JSON.stringify({
    id: 1, email: 'me@a.com', name: 'Me', phone: '', role: 'admin', status: 'active', mustChangePW: false,
  }));
  return render(
    <MemoryRouter>
      <AntApp>
        <AuthProvider>
          <AdminManagement />
        </AuthProvider>
      </AntApp>
    </MemoryRouter>,
  );
}

describe('AdminManagement', () => {
  beforeEach(async () => {
    getAdminsMock.mockReset();
    getAdminsMock.mockResolvedValue({ data: [], total: 0, page: 1, pageSize: 10 });
    createAdminMock.mockReset();
    deleteAdminMock.mockReset();
    localStorage.clear();
    // Clear any portal nodes leaked from antd Modal/message of prior tests
    document.body.innerHTML = '';
    await i18n.changeLanguage('en');
  });

  afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
  });

  it('lists admins fetched from API', async () => {
    getAdminsMock.mockResolvedValueOnce({ data: sampleAdmins, total: sampleAdmins.length, page: 1, pageSize: 10 });
    renderPage();
    expect(await screen.findByText('me@a.com')).toBeInTheDocument();
    expect(screen.getByText('other@a.com')).toBeInTheDocument();
  });

  it('disables the delete button for the current user (self-delete guard)', async () => {
    getAdminsMock.mockResolvedValueOnce({ data: sampleAdmins, total: sampleAdmins.length, page: 1, pageSize: 10 });
    renderPage();
    await screen.findByText('me@a.com');

    const meRow = screen.getByText('me@a.com').closest('tr')!;
    const meDeleteBtn = within(meRow).getByRole('button', { name: 'Delete' });
    expect(meDeleteBtn).toBeDisabled();

    const otherRow = screen.getByText('other@a.com').closest('tr')!;
    const otherDeleteBtn = within(otherRow).getByRole('button', { name: 'Delete' });
    expect(otherDeleteBtn).not.toBeDisabled();
  });

  it('opens create modal and shows password mismatch error', async () => {
    getAdminsMock.mockResolvedValueOnce({ data: sampleAdmins, total: sampleAdmins.length, page: 1, pageSize: 10 });
    renderPage();
    await screen.findByText('me@a.com');

    const user = userEvent.setup({ delay: null });
    await user.click(screen.getByRole('button', { name: /Add Admin/ }));

    // 用 fireEvent.change 直接設值，比 userEvent.type 在 antd Form 內快很多
    const nameInput = await screen.findByLabelText('Name');
    fireEvent.change(nameInput, { target: { value: 'New' } });
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'new@a.com' } });
    const passwordInputs = screen.getAllByLabelText(/Password/);
    fireEvent.change(passwordInputs[0], { target: { value: 'password1234' } });
    fireEvent.change(passwordInputs[1], { target: { value: 'different5678' } });

    await user.click(screen.getByRole('button', { name: 'Create' }));

    expect(await screen.findByText('Passwords do not match')).toBeInTheDocument();
    expect(createAdminMock).not.toHaveBeenCalled();
  });
});
