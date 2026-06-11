import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App as AntApp } from 'antd';
import TranslatorManagement from '../TranslatorManagement';
import i18n from '../../../i18n';

const getTranslatorsMock = vi.fn();
const createTranslatorMock = vi.fn();

vi.mock('../../../api/translators', () => ({
  getTranslators: () => getTranslatorsMock(),
  createTranslator: (data: unknown) => createTranslatorMock(data),
  updateTranslator: vi.fn(),
  disableTranslator: vi.fn(),
  resetTranslatorPassword: vi.fn(),
}));

function renderPage() {
  return render(
    <AntApp>
      <TranslatorManagement />
    </AntApp>,
  );
}

describe('TranslatorManagement', () => {
  beforeEach(async () => {
    getTranslatorsMock.mockReset();
    createTranslatorMock.mockReset();
    getTranslatorsMock.mockResolvedValue([]);
    document.body.innerHTML = '';
    await i18n.changeLanguage('en');
  });

  afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
  });

  it('shows the backend error message when creating a translator with a duplicate email', async () => {
    // Simulate the axios interceptor having attached a translated message for
    // the EMAIL_TAKEN code returned by the backend (HTTP 409).
    createTranslatorMock.mockRejectedValueOnce({
      response: { status: 409, data: { code: 'EMAIL_TAKEN', message: 'Email already in use' } },
    });
    renderPage();

    const user = userEvent.setup({ delay: null });
    await user.click(screen.getByRole('button', { name: /Add Translator/ }));

    fireEvent.change(await screen.findByLabelText('Name'), { target: { value: 'Dup' } });
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'dup@a.com' } });
    fireEvent.change(screen.getByLabelText('Phone'), { target: { value: '0900000000' } });
    fireEvent.change(screen.getByLabelText('Password'), { target: { value: 'secret123' } });

    await user.click(screen.getByRole('button', { name: 'Create' }));

    // The specific backend reason must surface, not a generic "Failed".
    expect(await screen.findByText('Email already in use')).toBeInTheDocument();
    expect(createTranslatorMock).toHaveBeenCalledOnce();
  });
});
