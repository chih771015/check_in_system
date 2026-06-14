import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App as AntApp } from 'antd';
import DiagnosisUploadModal from '../DiagnosisUploadModal';
import type { DiagnosisPhotoItem } from '../../api/checkins';
import i18n from '../../i18n';

const uploadMock = vi.fn();
const listPhotosMock = vi.fn();
const deletePhotoMock = vi.fn();

function makeFile(name: string) {
  return new File(['data'], name, { type: 'image/jpeg' });
}

// A file with a forced size (avoids allocating real megabytes in the test).
function bigFile(name: string, size: number) {
  const f = new File(['x'], name, { type: 'image/jpeg' });
  Object.defineProperty(f, 'size', { value: size });
  return f;
}

function setup(
  existing: DiagnosisPhotoItem[] = [],
  gate: { canUpload?: boolean; canDelete?: boolean } = {},
) {
  const { canUpload = true, canDelete = true } = gate;
  const onClose = vi.fn();
  const onUploaded = vi.fn();
  uploadMock.mockReset();
  listPhotosMock.mockReset();
  deletePhotoMock.mockReset();
  uploadMock.mockResolvedValue({ message: 'ok' });
  listPhotosMock.mockResolvedValue(existing);
  deletePhotoMock.mockResolvedValue({ message: 'ok' });
  render(
    <AntApp>
      <DiagnosisUploadModal
        open
        schedulePatientId={42}
        canUpload={canUpload}
        canDelete={canDelete}
        onClose={onClose}
        onUploaded={onUploaded}
        upload={uploadMock}
        listPhotos={listPhotosMock}
        deletePhoto={deletePhotoMock}
      />
    </AntApp>,
  );
  return { onClose, onUploaded };
}

describe('DiagnosisUploadModal', () => {
  beforeEach(async () => {
    document.body.innerHTML = '';
    await i18n.changeLanguage('en');
  });
  afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
  });

  it('renders manage title, fetches existing photos and disables submit with no selection', async () => {
    setup([]);
    expect(screen.getByText('Manage Diagnosis Photos')).toBeInTheDocument();
    expect(await screen.findByText('No photos uploaded yet')).toBeInTheDocument();
    expect(listPhotosMock).toHaveBeenCalledWith(42);
    const submit = screen.getByRole('button', { name: 'Submit' });
    expect(submit).toBeDisabled();
  });

  it('caps selection at the remaining slots (30) and warns', async () => {
    const { onUploaded } = setup([]);
    await screen.findByText('No photos uploaded yet');
    const user = userEvent.setup({ delay: null });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    // Pick 31 → capped to the 30-photo limit.
    const many = Array.from({ length: 31 }, (_, i) => makeFile(`p${i}.jpg`));
    await user.upload(input, many);

    expect(await screen.findByText(/up to 30/)).toBeInTheDocument();

    const submit = screen.getByRole('button', { name: 'Submit' });
    await user.click(submit);
    await vi.waitFor(() => expect(uploadMock).toHaveBeenCalledOnce());
    const filesArg = uploadMock.mock.calls[0][1] as File[];
    expect(filesArg).toHaveLength(30);
    await vi.waitFor(() => expect(onUploaded).toHaveBeenCalled());
  });

  it('blocks submit when the selected batch is too large (pre-empts nginx 413)', async () => {
    setup([]);
    await screen.findByText('No photos uploaded yet');
    const user = userEvent.setup({ delay: null });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    await user.upload(input, [bigFile('huge.jpg', 95 * 1024 * 1024)]);

    // Inline hint shown and submit disabled — no oversized request is sent.
    expect((await screen.findAllByText(/too large/i)).length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: 'Submit' })).toBeDisabled();
    expect(uploadMock).not.toHaveBeenCalled();
  });

  it('uploads 1-2 chosen files and notifies the parent (modal stays open)', async () => {
    const { onUploaded, onClose } = setup([]);
    await screen.findByText('No photos uploaded yet');
    const user = userEvent.setup({ delay: null });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;
    await user.upload(input, [makeFile('a.jpg'), makeFile('b.jpg')]);

    await user.click(screen.getByRole('button', { name: 'Submit' }));

    await vi.waitFor(() => expect(uploadMock).toHaveBeenCalledOnce());
    const [spID, files] = uploadMock.mock.calls[0];
    expect(spID).toBe(42);
    expect(files).toHaveLength(2);
    await vi.waitFor(() => expect(onUploaded).toHaveBeenCalled());
    // Re-fetches to refresh the existing list, and does NOT auto-close.
    expect(listPhotosMock).toHaveBeenCalledTimes(2);
    expect(onClose).not.toHaveBeenCalled();
  });

  it('shows existing photos and deletes one by id', async () => {
    const { onUploaded } = setup([
      { id: 7, photoUrl: '/uploads/x.jpg' },
      { id: 8, photoUrl: '/uploads/y.jpg' },
    ]);
    const user = userEvent.setup({ delay: null });

    // Two delete buttons, one per existing photo.
    const delButtons = await screen.findAllByRole('button', { name: 'Delete photo' });
    expect(delButtons).toHaveLength(2);

    await user.click(delButtons[0]);
    // Confirm in the popconfirm.
    await user.click(await screen.findByRole('button', { name: 'Confirm' }));

    await vi.waitFor(() => expect(deletePhotoMock).toHaveBeenCalledWith(7));
    await vi.waitFor(() => expect(onUploaded).toHaveBeenCalled());
  });

  it('view-only mode (no upload, no delete) shows photos but no edit affordances', async () => {
    setup([{ id: 7, photoUrl: '/uploads/x.jpg' }], { canUpload: false, canDelete: false });

    expect(await screen.findByText('View Photos')).toBeInTheDocument();
    expect(document.querySelector('input[type="file"]')).toBeNull();
    expect(screen.queryByRole('button', { name: 'Delete photo' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Submit' })).toBeNull();
  });

  it('append-only mode (upload yes, delete no) shows add input but no delete buttons', async () => {
    setup([{ id: 7, photoUrl: '/uploads/x.jpg' }], { canUpload: true, canDelete: false });

    // Append title; can add but not delete (translator after leave).
    expect(await screen.findByText('Add Photos (post-visit)')).toBeInTheDocument();
    expect(document.querySelector('input[type="file"]')).not.toBeNull();
    expect(screen.queryByRole('button', { name: 'Delete photo' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Submit' })).toBeInTheDocument();
  });
});
