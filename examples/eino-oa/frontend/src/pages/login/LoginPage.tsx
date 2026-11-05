import { useState } from 'react';
import { Button, Card, Form, Input, Typography, message } from 'antd';
import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { authApi } from '../../services/api';
import { useAuth } from '../../auth/context';

const { Text } = Typography;

interface LoginValues {
  username: string;
  password: string;
}

export default function LoginPage() {
  const { setUser } = useAuth();
  const [loading, setLoading] = useState(false);

  const onFinish = async (values: LoginValues) => {
    setLoading(true);
    try {
      const res = await authApi.login(values.username.trim(), values.password);
      setUser(res.data.user);

      // 支持 OAuth 授权页跳转过来时登录后回跳。
      // 只允许同源地址，避免 //evil.com、/\evil.com 之类的开放重定向。
      const redirect = new URLSearchParams(window.location.search).get('redirect');
      if (redirect) {
        try {
          const target = new URL(redirect, window.location.origin);
          if (target.origin === window.location.origin) {
            window.location.assign(target.pathname + target.search + target.hash);
            return;
          }
        } catch {
          // 非法 URL 直接回首页
        }
      }
      window.location.assign('/students');
    } catch {
      message.error('用户名或密码错误');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-page">
      <Card className="login-card" variant="borderless">
        <div className="login-brand">EINO OA</div>
        <h1 className="login-title">大学生信息管理系统</h1>
        <Text type="secondary">登录后管理学生、课程、成绩，或授权外部 Agent 访问。</Text>

        <Form<LoginValues>
          layout="vertical"
          size="large"
          onFinish={onFinish}
          style={{ marginTop: 28 }}
          requiredMark={false}
        >
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input prefix={<UserOutlined />} placeholder="admin" autoComplete="username" autoFocus />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="请输入密码"
              autoComplete="current-password"
            />
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={loading}>
            登录
          </Button>
        </Form>

        <p className="login-foot">
          账号密码由服务端 <code>backend/.env</code> 配置：
          <code> ADMIN_USERNAME </code> / <code> ADMIN_PASSWORD </code>，修改后重启服务生效。
        </p>
      </Card>
    </div>
  );
}
