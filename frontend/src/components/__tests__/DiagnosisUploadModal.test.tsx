import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App as AntApp } from 'antd';
import DiagnosisUploadModal from '../DiagnosisUploadModal';

const uploadMock = vi.fn();

function makeFile(name: string) {
  return new File(['data'], name, { type: 'image/jpeg' });
}

function setup(open = true) {
  const onClose = vi.fn();
  const onUploaded = vi.fn();
  uploadMock.mockReset();
  uploadMock.mockResolvedValue({ message: 'ok' });
  render(
    <AntApp>
      <DiagnosisUploadModal
        open={open}
        schedulePatientId={42}
        onClose={onClose}
        onUploaded={onUploaded}
        upload={uploadMock}
      />
    </AntApp>,
  );
  return { onClose, onUploaded };
}

describe('DiagnosisUploadModal', () => {
  beforeEach(() => {
    document.body.innerHTML = '';
  });
  afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
  });

  it('renders title and submit disabled when no files selected', () => {
    setup();
    expect(screen.getByText(/Upload Diagnosis|上傳診斷證明|อัปโหลด/)).toBeInTheDocument();
    const submit = screen.getByRole('button', { name: /Submit|送出|ส่ง/ });
    expect(submit).toBeDisabled();
  });

  it('caps selection at 3 files and warns the user (does not block submit)', async () => {
    const { onUploaded } = setup();
    const user = userEvent.setup({ delay: null });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    await user.upload(input, [makeFile('a.jpg'), makeFile('b.jpg'), makeFile('c.jpg'), makeFile('d.jpg')]);

    // antd App.message renders a warning toast
    expect(await screen.findByText(/up to 3|最多.*3|สูงสุด 3/)).toBeInTheDocument();
    // Submit should be enabled — we kept the first 3 instead of clearing.
    const submit = screen.getByRole('button', { name: /Submit|送出|ส่ง/ });
    expect(submit).toBeEnabled();

    await user.click(submit);
    await vi.waitFor(() => expect(uploadMock).toHaveBeenCalledOnce());
    const filesArg = uploadMock.mock.calls[0][1] as File[];
    expect(filesArg).toHaveLength(3);
    await vi.waitFor(() => expect(onUploaded).toHaveBeenCalled());
  });

  it('enables submit and calls upload when 1-3 files chosen', async () => {
    const { onUploaded, onClose } = setup();
    const user = userEvent.setup({ delay: null });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    await user.upload(input, [makeFile('a.jpg'), makeFile('b.jpg')]);

    const submit = screen.getByRole('button', { name: /Submit|送出|ส่ง/ });
    expect(submit).toBeEnabled();

    await user.click(submit);

    await vi.waitFor(() => expect(uploadMock).toHaveBeenCalledOnce());
    const [spID, files] = uploadMock.mock.calls[0];
    expect(spID).toBe(42);
    expect(files).toHaveLength(2);
    await vi.waitFor(() => expect(onUploaded).toHaveBeenCalledOnce());
    expect(onClose).toHaveBeenCalled();
  });
});
