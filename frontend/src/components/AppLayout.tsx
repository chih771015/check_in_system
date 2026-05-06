import { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Button, Typography, theme, Modal, Form, Input, App } from 'antd';
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
} from '@ant-design/icons';
import { useAuth } from '../stores/authStore';
import { changePassword } from '../api/auth';

const { Header, Sider, Content } = Layout;

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const [changePWOpen, setChangePWOpen] = useState(false);
  const [changePWLoading, setChangePWLoading] = useState(false);
  const [changePWForm] = Form.useForm();
  const { user, login, logout, isAdmin } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { token: themeToken } = theme.useToken();
  const { message } = App.useApp();

  const handleChangePW = async (values: {
    oldPassword: string;
    newPassword: string;
    confirmPassword: string;
  }) => {
    if (values.newPassword !== values.confirmPassword) {
      void message.error('兩次輸入的密碼不一致');
      return;
    }
    setChangePWLoading(true);
    try {
      const res = await changePassword(values.oldPassword, values.newPassword);
      // Update stored token so the session continues without re-login
      if (user) login(res.token, { ...user, mustChangePW: false });
      void message.success('密碼已更新');
      setChangePWOpen(false);
      changePWForm.resetFields();
    } catch {
      void message.error('舊密碼錯誤或更新失敗');
    } finally {
      setChangePWLoading(false);
    }
  };

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const adminMenuItems = [
    {
      key: '/admin/translators',
      icon: <TeamOutlined />,
      label: '翻譯員管理',
    },
    {
      key: '/admin/schedules',
      icon: <ScheduleOutlined />,
      label: '排班管理',
    },
    {
      key: '/admin/checkins',
      icon: <CheckSquareOutlined />,
      label: '打卡紀錄',
    },
    {
      key: '/admin/export-settings',
      icon: <SettingOutlined />,
      label: '定期匯出設定',
    },
    {
      key: '/admin/audit-logs',
      icon: <FileSearchOutlined />,
      label: '操作紀錄',
    },
    {
      key: '/admin/admins',
      icon: <UserSwitchOutlined />,
      label: '帳號管理',
    },
  ];

  const translatorMenuItems = [
    {
      key: '/my-schedules',
      icon: <CalendarOutlined />,
      label: '我的排班',
    },
    {
      key: '/my-checkins',
      icon: <HistoryOutlined />,
      label: '我的打卡紀錄',
    },
  ];

  const menuItems = isAdmin ? adminMenuItems : translatorMenuItems;

  return (
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
              翻譯員打卡系統
            </Typography.Text>
          )}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
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
            <Typography.Text>{user?.name}</Typography.Text>
            {isAdmin && (
              <Button type="text" icon={<LockOutlined />} onClick={() => setChangePWOpen(true)}>
                修改密碼
              </Button>
            )}
            <Button type="text" icon={<LogoutOutlined />} onClick={handleLogout}>
              登出
            </Button>
          </div>
        </Header>
        <Content style={{ margin: 16, padding: 24, background: themeToken.colorBgContainer, borderRadius: 8, overflow: 'auto' }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>

      {/* Change password modal — admin only */}
      <Modal
        title="修改密碼"
        open={changePWOpen}
        onCancel={() => { setChangePWOpen(false); changePWForm.resetFields(); }}
        footer={null}
        destroyOnClose
      >
        <Form form={changePWForm} layout="vertical" onFinish={handleChangePW} style={{ marginTop: 8 }}>
          <Form.Item name="oldPassword" label="舊密碼" rules={[{ required: true, message: '請輸入舊密碼' }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label="新密碼"
            rules={[
              { required: true, message: '請輸入新密碼' },
              { min: 8, message: '密碼至少 8 個字元' },
            ]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="確認新密碼"
            rules={[{ required: true, message: '請再次輸入新密碼' }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" block loading={changePWLoading}>
              確認修改
            </Button>
          </Form.Item>
        </Form>
      </Modal>
  );
}
