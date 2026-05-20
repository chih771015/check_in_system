import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { App as AntApp } from 'antd';
import ChangePasswordPage from '../ChangePassword';
import { AuthProvider } from '../../stores/authStore';
import i18n from '../../i18n';

const changePasswordMock = vi.fn();
vi.mock('../../api/auth', () => ({
  changePassword: (...args: unknown[]) => changePasswordMock(...args),
}));

const navigateMock = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return { ...actual, useNavigate: () => navigateMock };
});

function renderPage() {
  return render(
    <MemoryRouter>
      <AntApp>
        <AuthProvider>
          <ChangePasswordPage />
        </AuthProvider>
      </AntApp>
    </MemoryRouter>,
  );
}

describe('ChangePasswordPage', () => {
  beforeEach(async () => {
    changePasswordMock.mockReset();
    navigateMock.mockReset();
    localStorage.clear();
    document.body.innerHTML = '';
    // Seed an admin user so the page has someone to update
    localStorage.setItem('token', 'tk');
    localStorage.setItem('user', JSON.stringify({
      id: 1, email: 'a@a', name: 'A', phone: '', role: 'admin', status: 'active', mustChangePW: true,
    }));
    await i18n.changeLanguage('en');
  });

  afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
  });

  it('renders required fields in English', () => {
    renderPage();
    expect(screen.getByText('Change Password')).toBeInTheDocument();
    expect(screen.getAllByText(/Old Password|New Password|Confirm Password/i).length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: /Update Password/i })).toBeInTheDocument();
  });

  it('shows mismatch error when new and confirm differ', async () => {
    renderPage();
    const user = userEvent.setup({ delay: null });
    const inputs = screen.getAllByPlaceholderText(/Password/);
    fireEvent.change(inputs[0], { target: { value: 'oldpass' } });
    fireEvent.change(inputs[1], { target: { value: 'newpass1234' } });
    fireEvent.change(inputs[2], { target: { value: 'different' } });
    await user.click(screen.getByRole('button', { name: /Update Password/i }));

    expect(await screen.findByText('Passwords do not match')).toBeInTheDocument();
    expect(changePasswordMock).not.toHaveBeenCalled();
  });

  it('calls API and navigates to admin landing on success for admin user', async () => {
    changePasswordMock.mockResolvedValueOnce({ token: 'newtk' });
    renderPage();
    const user = userEvent.setup({ delay: null });
    const inputs = screen.getAllByPlaceholderText(/Password/);
    fireEvent.change(inputs[0], { target: { value: 'oldpass' } });
    fireEvent.change(inputs[1], { target: { value: 'newpass1234' } });
    fireEvent.change(inputs[2], { target: { value: 'newpass1234' } });
    await user.click(screen.getByRole('button', { name: /Update Password/i }));

    await vi.waitFor(() => {
      expect(changePasswordMock).toHaveBeenCalledWith('oldpass', 'newpass1234');
      expect(navigateMock).toHaveBeenCalledWith('/admin/translators');
    });
  });

  it('shows translated error toast when API fails', async () => {
    changePasswordMock.mockRejectedValueOnce(new Error('boom'));
    renderPage();
    const user = userEvent.setup({ delay: null });
    const inputs = screen.getAllByPlaceholderText(/Password/);
    fireEvent.change(inputs[0], { target: { value: 'wrong' } });
    fireEvent.change(inputs[1], { target: { value: 'newpass1234' } });
    fireEvent.change(inputs[2], { target: { value: 'newpass1234' } });
    await user.click(screen.getByRole('button', { name: /Update Password/i }));

    expect(await screen.findByText('Old password is incorrect')).toBeInTheDocument();
    expect(navigateMock).not.toHaveBeenCalled();
  });
});
