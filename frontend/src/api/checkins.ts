import client from './client';

export function checkin(formData: FormData) {
  return client
    .post('/checkins', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data);
}

export function makeupCheckin(formData: FormData) {
  return client
    .post('/checkins/makeup', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    .then((r) => r.data);
}
