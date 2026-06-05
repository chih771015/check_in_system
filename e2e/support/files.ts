/**
 * Shared file fixtures for upload flows (checkin selfie, diagnosis photos).
 *
 * We build the image in-memory rather than committing a binary blob so the
 * repo stays text-only and the bytes are obviously a 1x1 JPEG.
 */

// A minimal valid 1x1 white JPEG.
const SELFIE_JPEG_BASE64 =
  '/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRof' +
  'Hh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/wAALCAABAAEBAREA/8QAFAAB' +
  'AAAAAAAAAAAAAAAAAAAACf/EABQQAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQEAAD8AfwD/2Q==';

export interface UploadFile {
  name: string;
  mimeType: string;
  buffer: Buffer;
}

/** A throwaway selfie image for checkin / makeup uploads. */
export function selfieFile(name = 'selfie.jpg'): UploadFile {
  return {
    name,
    mimeType: 'image/jpeg',
    buffer: Buffer.from(SELFIE_JPEG_BASE64, 'base64'),
  };
}
