import { describe, it, expect } from 'vitest';
import { parseAuditDetail, diffFields } from './auditDetail';

describe('parseAuditDetail', () => {
  it('returns empty text for empty detail', () => {
    expect(parseAuditDetail('')).toEqual({ kind: 'text', text: '' });
  });

  it('treats non-JSON as plain text', () => {
    expect(parseAuditDetail('email=a@b.com name=Foo')).toEqual({
      kind: 'text',
      text: 'email=a@b.com name=Foo',
    });
  });

  it('treats a JSON object without before/after as plain text', () => {
    const s = '{"foo":1}';
    expect(parseAuditDetail(s)).toEqual({ kind: 'text', text: s });
  });

  it('parses a delete change-set (before only)', () => {
    const parsed = parseAuditDetail('{"before":{"id":1,"name":"X"}}');
    expect(parsed).toEqual({
      kind: 'change',
      before: { id: 1, name: 'X' },
      after: undefined,
    });
  });

  it('parses an update change-set (before + after)', () => {
    const parsed = parseAuditDetail(
      '{"before":{"name":"Old"},"after":{"name":"New"}}',
    );
    expect(parsed).toEqual({
      kind: 'change',
      before: { name: 'Old' },
      after: { name: 'New' },
    });
  });
});

describe('diffFields', () => {
  it('flags only the fields that changed', () => {
    const fields = diffFields(
      { name: 'Old', phone: '111' },
      { name: 'New', phone: '111' },
    );
    const name = fields.find((f) => f.key === 'name');
    const phone = fields.find((f) => f.key === 'phone');
    expect(name?.changed).toBe(true);
    expect(name?.before).toBe('Old');
    expect(name?.after).toBe('New');
    expect(phone?.changed).toBe(false);
  });

  it('includes keys present on only one side', () => {
    const fields = diffFields({ a: 1 }, { b: 2 });
    expect(fields.map((f) => f.key).sort()).toEqual(['a', 'b']);
    expect(fields.every((f) => f.changed)).toBe(true);
  });

  it('handles undefined before (delete) gracefully', () => {
    const fields = diffFields(undefined, undefined);
    expect(fields).toEqual([]);
  });
});
