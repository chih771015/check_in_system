import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { App as AntApp } from 'antd';
import LoginPage from '../Login';
import { AuthProvider } from '../../stores/authStore';
import i18n from '../../i18n';

// Mock the login API so tests don't touch network.
const loginMock = vi.fn();
vi.mock('../../api/auth', () => ({
  login: (...args: unknown[]) => loginMock(...args),
}));

// Mock useNavigate so we can assert routing decisions.
const navigateMock = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return { ...actual, useNavigate: () => navigateMock };
});

function renderLogin() {
  return render(
    <MemoryRouter>
      <AntApp>
        <AuthProvider>
          <LoginPage />
        </AuthProvider>
      </AntApp>
    </MemoryRouter>,
  );
}

describe('LoginPage', () => {
  beforeEach(async () => {
    loginMock.mockReset();
    navigateMock.mockReset();
    localStorage.clear();
    document.body.innerHTML = '';
    await i18n.changeLanguage('en');
  });

  afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
  });

  it('renders English placeholders by default', () => {
    renderLogin();
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign In' })).toBeInTheDocument();
  });

  it('renders Chinese placeholders after switching to zh-TW', async () => {
    await i18n.changeLanguage('zh-TW');
    renderLogin();
    expect(screen.getByPlaceholderText('電子郵件')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('密碼')).toBeInTheDocument();
  });

  it('navigates to /admin/translators after admin login', async () => {
    loginMock.mockResolvedValueOnce({
      token: 'tk',
      user: { id: 1, email: 'a@a', name: 'A', phone: '', role: 'admin', status: 'active', mustChangePW: false },
    });
    renderLogin();

    const user = userEvent.setup({ delay: null });
    fireEvent.change(screen.getByPlaceholderText('Email'), { target: { value: 'a@a.com' } });
    fireEvent.change(screen.getByPlaceholderText('Password'), { target: { value: 'pass' } });
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    await vi.waitFor(() => {
      expect(navigateMock).toHaveBeenCalledWith('/admin/translators');
    });
  });

  it('navigates to /change-password when mustChangePW is true', async () => {
    loginMock.mockResolvedValueOnce({
      token: 'tk',
      user: { id: 1, email: 'a@a', name: 'A', phone: '', role: 'admin', status: 'active', mustChangePW: true },
    });
    renderLogin();

    const user = userEvent.setup({ delay: null });
    fireEvent.change(screen.getByPlaceholderText('Email'), { target: { value: 'a@a.com' } });
    fireEvent.change(screen.getByPlaceholderText('Password'), { target: { value: 'pass' } });
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    await vi.waitFor(() => {
      expect(navigateMock).toHaveBeenCalledWith('/change-password');
    });
  });

  it('navigates to /my-schedules after translator login', async () => {
    loginMock.mockResolvedValueOnce({
      token: 'tk',
      user: { id: 2, email: 't@t', name: 'T', phone: '', role: 'translator', status: 'active', mustChangePW: false },
    });
    renderLogin();

    const user = userEvent.setup({ delay: null });
    fireEvent.change(screen.getByPlaceholderText('Email'), { target: { value: 't@t.com' } });
    fireEvent.change(screen.getByPlaceholderText('Password'), { target: { value: 'pass' } });
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    await vi.waitFor(() => {
      expect(navigateMock).toHaveBeenCalledWith('/my-schedules');
    });
  });

  it('shows translated error toast on failed login', async () => {
    loginMock.mockRejectedValueOnce(new Error('boom'));
    renderLogin();

    const user = userEvent.setup({ delay: null });
    fireEvent.change(screen.getByPlaceholderText('Email'), { target: { value: 'a@a.com' } });
    fireEvent.change(screen.getByPlaceholderText('Password'), { target: { value: 'wrong' } });
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    // antd App.message renders a transient toast; query its content
    expect(await screen.findByText('Invalid email or password')).toBeInTheDocument();
    expect(navigateMock).not.toHaveBeenCalled();
  });
});
