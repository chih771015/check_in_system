import { Routes, Route, Navigate } from 'react-router-dom';
import { App as AntApp, ConfigProvider } from 'antd';
import enUS from 'antd/locale/en_US';
import zhTW from 'antd/locale/zh_TW';
import thTH from 'antd/locale/th_TH';
import { useTranslation } from 'react-i18next';
import { AuthProvider, useAuth } from './stores/authStore';
import AppLayout from './components/AppLayout';
import LoginPage from './pages/Login';
import ChangePasswordPage from './pages/ChangePassword';
import TranslatorManagement from './pages/admin/TranslatorManagement';
import AdminManagement from './pages/admin/AdminManagement';
import PatientManagement from './pages/admin/PatientManagement';
import PatientHistory from './pages/admin/PatientHistory';
import ScheduleManagement from './pages/admin/ScheduleManagement';
import CheckinRecords from './pages/admin/CheckinRecords';
import DiagnosisResultsPage from './pages/admin/DiagnosisResults';
import ExportSettings from './pages/admin/ExportSettings';
import AuditLogs from './pages/admin/AuditLogs';
import MySchedules from './pages/translator/MySchedules';
import MyCheckinsPage from './pages/translator/MyCheckins';
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
          path="/admin/diagnosis-results"
          element={
            <RequireAdmin>
              <DiagnosisResultsPage />
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
          path="/admin/audit-logs"
          element={
            <RequireAdmin>
              <AuditLogs />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/patients"
          element={
            <RequireAdmin>
              <PatientManagement />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/patients/:id/history"
          element={
            <RequireAdmin>
              <PatientHistory />
            </RequireAdmin>
          }
        />
        <Route
          path="/admin/admins"
          element={
            <RequireAdmin>
              <AdminManagement />
            </RequireAdmin>
          }
        />
        <Route
          path="/my-checkins"
          element={
            <RequireTranslator>
              <MyCheckinsPage />
            </RequireTranslator>
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

const antdLocaleMap = {
  en: enUS,
  'zh-TW': zhTW,
  th: thTH,
} as const;

export default function App() {
  const { i18n } = useTranslation();
  const lang = (i18n.language || 'en') as keyof typeof antdLocaleMap;
  const antdLocale = antdLocaleMap[lang] ?? enUS;
  return (
    <ConfigProvider locale={antdLocale}>
      <AntApp>
        <AuthProvider>
          <AppRoutes />
        </AuthProvider>
      </AntApp>
    </ConfigProvider>
  );
}
