import { describe, it, expect } from 'vitest';
import { formatNT } from './money';

describe('formatNT', () => {
  it('formats an integer with thousands separators and no currency prefix', () => {
    expect(formatNT(12345)).toBe('12,345');
    expect(formatNT(0)).toBe('0');
  });

  it('coalesces null/undefined to 0', () => {
    expect(formatNT(undefined)).toBe('0');
    expect(formatNT(null)).toBe('0');
  });
});
