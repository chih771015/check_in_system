import { useEffect, useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Button, Typography, theme, Modal, Form, Input, App, Select, Grid } from 'antd';
import { useTranslation } from 'react-i18next';
import { setLanguage, SUPPORTED_LANGUAGES } from '../i18n';
import {
  TeamOutlined,
  ScheduleOutlined,
  CalendarOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  CheckSquareOutlined,
  SettingOutlined,
  FileSearchOutlined,
  HistoryOutlined,
  LockOutlined,
  UserSwitchOutlined,
  IdcardOutlined,
  ProfileOutlined,
} from '@ant-design/icons';
import { useAuth } from '../stores/authStore';
import { changePassword } from '../api/auth';
import { getMonthlyTotal } from '../api/stats';

const { Header, Sider, Content } = Layout;

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const [changePWOpen, setChangePWOpen] = useState(false);
  const [changePWLoading, setChangePWLoading] = useState(false);
  const [changePWForm] = Form.useForm();
  const { user, login, logout, isAdmin } = useAuth();
  // Admin-only banner: current-month total patient expenditure (NT$). Shown
  // atop every admin page and re-fetched on each navigation so it reflects
  // actual-amount edits (and a crossed month boundary) without a full reload.
  const [monthlyTotal, setMonthlyTotal] = useState<{ yearMonth: string; total: number } | null>(null);
  const navigate = useNavigate();
  const location = useLocation();
  const { token: themeToken } = theme.useToken();
  const { message } = App.useApp();
  const { t, i18n } = useTranslation();
  // Below `lg` (992px) we treat the sider as a drawer-style overlay and
  // auto-collapse after navigation so users on phones don't have to tap
  // the hamburger to dismiss it every time.
  const screens = Grid.useBreakpoint();

  useEffect(() => {
    if (!isAdmin) return;
    let active = true;
    getMonthlyTotal()
      .then((r) => { if (active) setMonthlyTotal(r); })
      .catch(() => { /* banner is best-effort; ignore errors */ });
    return () => { active = false; };
  }, [isAdmin, location.pathname]);

  const handleChangePW = async (values: {
    oldPassword: string;
    newPassword: string;
    confirmPassword: string;
  }) => {
    if (values.newPassword !== values.confirmPassword) {
      void message.error(t('changePassword.mismatch'));
      return;
    }
    setChangePWLoading(true);
    try {
      const res = await changePassword(values.oldPassword, values.newPassword);
      if (user) login(res.token, { ...user, mustChangePW: false });
      void message.success(t('changePassword.successUpdated'));
      setChangePWOpen(false);
      changePWForm.resetFields();
    } catch {
      void message.error(t('errors.OLD_PASSWORD_INCORRECT'));
    } finally {
      setChangePWLoading(false);
    }
  };

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const adminMenuItems = [
    { key: '/admin/translators', icon: <TeamOutlined />, label: t('nav.translators') },
    { key: '/admin/schedules', icon: <ScheduleOutlined />, label: t('nav.schedules') },
    { key: '/admin/patients', icon: <IdcardOutlined />, label: t('nav.patients') },
    { key: '/admin/checkins', icon: <CheckSquareOutlined />, label: t('nav.checkins') },
    { key: '/admin/diagnosis-results', icon: <ProfileOutlined />, label: t('nav.diagnosisResults') },
    { key: '/admin/export-settings', icon: <SettingOutlined />, label: t('nav.exportSettings') },
    { key: '/admin/audit-logs', icon: <FileSearchOutlined />, label: t('nav.auditLogs') },
    { key: '/admin/admins', icon: <UserSwitchOutlined />, label: t('nav.admins') },
  ];

  const translatorMenuItems = [
    { key: '/my-schedules', icon: <CalendarOutlined />, label: t('nav.mySchedules') },
    { key: '/my-checkins', icon: <HistoryOutlined />, label: t('nav.myCheckins') },
  ];

  const menuItems = isAdmin ? adminMenuItems : translatorMenuItems;

  return (
    <>
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={setCollapsed}
        breakpoint="lg"
        collapsedWidth={0}
        trigger={null}
        style={{ background: themeToken.colorBgContainer }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            borderBottom: `1px solid ${themeToken.colorBorderSecondary}`,
          }}
        >
          {!collapsed && (
            <Typography.Text strong style={{ fontSize: 14 }}>
              {t('app.title')}
            </Typography.Text>
          )}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => {
            navigate(key);
            // Mobile / tablet (< lg): collapse the sider after picking a route.
            if (!screens.lg) setCollapsed(true);
          }}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: '0 16px',
            background: themeToken.colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: `1px solid ${themeToken.colorBorderSecondary}`,
          }}
        >
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
          />
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Select
              size="small"
              value={i18n.language}
              style={{ width: 110 }}
              onChange={(v) => setLanguage(v as (typeof SUPPORTED_LANGUAGES)[number])}
              options={SUPPORTED_LANGUAGES.map((l) => ({ value: l, label: t(`language.${l}`) }))}
            />
            <Typography.Text>{user?.name}</Typography.Text>
            {isAdmin && (
              <Button type="text" icon={<LockOutlined />} onClick={() => setChangePWOpen(true)}>
                {t('common.changePassword')}
              </Button>
            )}
            <Button type="text" icon={<LogoutOutlined />} onClick={handleLogout}>
              {t('common.logout')}
            </Button>
          </div>
        </Header>
        {isAdmin && monthlyTotal && (
          <div
            style={{
              margin: '16px 16px 0',
              padding: '10px 16px',
              background: themeToken.colorPrimaryBg,
              border: `1px solid ${themeToken.colorPrimaryBorder}`,
              borderRadius: 8,
              display: 'flex',
              alignItems: 'baseline',
              gap: 8,
              flexWrap: 'wrap',
            }}
          >
            <Typography.Text type="secondary">
              {t('dashboard.monthlyTotal', { month: monthlyTotal.yearMonth })}
            </Typography.Text>
            <Typography.Text strong style={{ fontSize: 18 }}>
              NT$ {monthlyTotal.total.toLocaleString()}
            </Typography.Text>
          </div>
        )}
        <Content style={{ margin: 16, padding: 24, background: themeToken.colorBgContainer, borderRadius: 8, overflow: 'auto' }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>

      {/* Change password modal — admin only */}
      <Modal
        title={t('common.changePassword')}
        open={changePWOpen}
        onCancel={() => { setChangePWOpen(false); changePWForm.resetFields(); }}
        footer={null}
        destroyOnClose
      >
        <Form form={changePWForm} layout="vertical" onFinish={handleChangePW} style={{ marginTop: 8 }}>
          <Form.Item name="oldPassword" label={t('common.oldPassword')} rules={[{ required: true }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label={t('common.newPassword')}
            rules={[
              { required: true },
              { min: 8, message: t('changePassword.minLength') },
            ]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item name="confirmPassword" label={t('common.confirmPassword')} rules={[{ required: true }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" block loading={changePWLoading}>
              {t('changePassword.submit')}
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
