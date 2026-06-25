import { describe, it, expect } from 'vitest';
import { formatNT } from './money';

describe('formatNT', () => {
  it('formats an integer with thousands separators and a NT$ prefix', () => {
    expect(formatNT(12345)).toBe('NT$ 12,345');
    expect(formatNT(0)).toBe('NT$ 0');
  });

  it('coalesces null/undefined to 0', () => {
    expect(formatNT(undefined)).toBe('NT$ 0');
    expect(formatNT(null)).toBe('NT$ 0');
  });
});
