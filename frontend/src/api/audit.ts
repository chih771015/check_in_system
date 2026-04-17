import client from './client';

export interface AuditLog {
  id: number;
  admin_id: number;
  admin_name: string;
  action: string;
  target_type: string;
  target_id: number;
  detail: string;
  created_at: string;
}

export interface AuditLogListResponse {
  data: AuditLog[];
  total: number;
  page: number;
  pageSize: number;
}

export function getAuditLogs(params?: Record<string, string | number>) {
  return client
    .get<AuditLogListResponse>('/admin/audit-logs', { params })
    .then((r) => r.data);
}
