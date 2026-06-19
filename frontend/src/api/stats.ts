import client from './client';

/** Current-month actual-paid total across all patients (admin banner). */
export function getMonthlyTotal() {
  return client
    .get<{ yearMonth: string; total: number }>('/admin/stats/monthly-total')
    .then((r) => r.data);
}
