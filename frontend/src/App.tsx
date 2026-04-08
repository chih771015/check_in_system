import { Routes, Route, Navigate } from 'react-router-dom';
import { App as AntApp } from 'antd';
import { AuthProvider, useAuth } from './stores/authStore';
import AppLayout from './components/AppLayout';
import LoginPage from './pages/Login';
import ChangePasswordPage from './pages/ChangePassword';
import TranslatorManagement from './pages/admin/TranslatorManagement';
import ScheduleManagement from './pages/admin/ScheduleManagement';
import CheckinRecords from './pages/admin/CheckinRecords';
import ExportSettings from './pages/admin/ExportSettings';
import MySchedules from './pages/translator/MySchedules';
import CheckInPage from './pages/translator/CheckIn';
import MakeupCheckInPage from './pages/translator/MakeupCheckIn';
import type { ReactNode } from 'react';

function RequireAuth({ children }: { children: ReactNode }) {
  const { token, user } = useAuth();
  if (!token || !user) return <Navigate to="/login" replace />;
  if (user.mustChangePW) return <Navigate to="/change-password" replace />;
  return <>{children}</>;
}

function RequireAdmin({ children }: { children: ReactNode }) {
  const { isAdmin } = useAuth();
  if (!isAdmin) return <Navigate to="/my-schedules" replace />;
  return <>{children}</>;
}

function RequireTranslator({ children }: { children: ReactNode }) {
  const { isTranslator } = useAuth();
  if (!isTranslator) return <Navigate to="/admin/translators" replace />;
  return <>{children}</>;
}

function DefaultRedirect() {
  const { token, user } = useAuth();
  if (!token || !user) return <Navigate to="/login" replace />;
  if (user.mustChangePW) return <Navigate to="/change-password" replace />;
  if (user.role === 'admin') return <Navigate to="/admin/translators" replace />;
  return <Navigate to="/my-schedules" replace />;
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/change-password" element={<ChangePasswordPage />} />
      <Route
        element={
          <RequireAuth>
            <AppLayout />
          </RequireAuth>
        }
      >
        <Route
          path="/admin/translators"
          element={
            <RequireAdmin>
              <TranslatorManagement />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/schedules"
          element={
            <RequireAdmin>
              <ScheduleManagement />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/checkins"
          element={
            <RequireAdmin>
              <CheckinRecords />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/export-settings"
          element={
            <RequireAdmin>
              <ExportSettings />
            </RequireAdmin>
          }
        />
        <Route
          path="/my-schedules"
          element={
            <RequireTranslator>
              <MySchedules />
            </RequireTranslator>
          }
        />
        <Route
          path="/checkin/:scheduleId/:type"
          element={
            <RequireTranslator>
              <CheckInPage />
            </RequireTranslator>
          }
        />
        <Route
          path="/makeup/:scheduleId/:type"
          element={
            <RequireTranslator>
              <MakeupCheckInPage />
            </RequireTranslator>
          }
        />
      </Route>
      <Route path="*" element={<DefaultRedirect />} />
    </Routes>
  );
}

export default function App() {
  return (
    <AntApp>
      <AuthProvider>
        <AppRoutes />
      </AuthProvider>
    </AntApp>
  );
}
