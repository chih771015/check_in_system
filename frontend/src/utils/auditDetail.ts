// Audit-log detail parsing.
//
// The backend stores AuditLog.detail either as a plain string (legacy / simple
// events like imports) or as a JSON change-set produced by the service layer:
//   {"before": {...}, "after": {...}}
// "after" is omitted for deletes. These helpers normalise both shapes so the UI
// can render a readable before/after view.

export type AuditDetailParsed =
  | { kind: 'text'; text: string }
  | {
      kind: 'change';
      before?: Record<string, unknown>;
      after?: Record<string, unknown>;
    };

export function parseAuditDetail(detail: string): AuditDetailParsed {
  if (!detail) return { kind: 'text', text: '' };
  try {
    const obj = JSON.parse(detail);
    if (
      obj &&
      typeof obj === 'object' &&
      !Array.isArray(obj) &&
      ('before' in obj || 'after' in obj)
    ) {
      return {
        kind: 'change',
        before: obj.before ?? undefined,
        after: obj.after ?? undefined,
      };
    }
  } catch {
    // Not JSON — fall through to plain text.
  }
  return { kind: 'text', text: detail };
}

export interface FieldChange {
  key: string;
  before?: unknown;
  after?: unknown;
  changed: boolean;
}

// diffFields builds a per-field view over before/after, flagging which values
// actually changed (so an update can highlight only the modified fields).
export function diffFields(
  before?: Record<string, unknown>,
  after?: Record<string, unknown>,
): FieldChange[] {
  const keys = new Set<string>([
    ...Object.keys(before ?? {}),
    ...Object.keys(after ?? {}),
  ]);
  const out: FieldChange[] = [];
  for (const key of keys) {
    const b = before?.[key];
    const a = after?.[key];
    out.push({
      key,
      before: b,
      after: a,
      changed: JSON.stringify(b) !== JSON.stringify(a),
    });
  }
  return out;
}
