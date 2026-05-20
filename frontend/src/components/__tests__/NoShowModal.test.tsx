import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App as AntApp } from 'antd';
import NoShowModal from '../NoShowModal';

const markMock = vi.fn();

function setup() {
  markMock.mockReset();
  markMock.mockResolvedValue({});
  const onClose = vi.fn();
  const onDone = vi.fn();
  render(
    <AntApp>
      <NoShowModal
        open
        schedulePatientId={7}
        onClose={onClose}
        onDone={onDone}
        markNoShow={markMock}
      />
    </AntApp>,
  );
  return { onClose, onDone };
}

describe('NoShowModal', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
  });
  afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
  });

  it('submit disabled when reason empty', () => {
    setup();
    const submit = screen.getByRole('button', { name: /Submit|送出|ส่ง/ });
    expect(submit).toBeDisabled();
  });

  it('enables submit and calls markNoShow when reason filled', async () => {
    const { onDone, onClose } = setup();
    const user = userEvent.setup({ delay: null });

    const textarea = screen.getByRole('textbox');
    await user.type(textarea, 'patient called to cancel');

    const submit = screen.getByRole('button', { name: /Submit|送出|ส่ง/ });
    expect(submit).toBeEnabled();

    await user.click(submit);
    await vi.waitFor(() => expect(markMock).toHaveBeenCalledOnce());
    expect(markMock).toHaveBeenCalledWith(7, 'patient called to cancel');
    await vi.waitFor(() => expect(onDone).toHaveBeenCalledOnce());
    expect(onClose).toHaveBeenCalled();
  });
});
