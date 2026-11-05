import { useCallback, useEffect, useState } from 'react';
import {
  Button, Card, Form, Input, message, Modal, Select, Space, Table, Tag, Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { CopyOutlined, KeyOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { tokenApi, type AccessToken } from '../../services/api';

const { Paragraph, Text } = Typography;

interface CreateValues {
  name: string;
  scopes: string;
  expires_in_days: number;
}

function fmtTime(value?: string | null) {
  if (!value) return '—';
  return new Date(value).toLocaleString('zh-CN');
}

function statusOf(token: AccessToken) {
  if (token.revoked_at) return <Tag>已吊销</Tag>;
  if (token.expires_at && new Date(token.expires_at) < new Date()) return <Tag color="orange">已过期</Tag>;
  return <Tag color="green">有效</Tag>;
}

function buildInstruction(token: string) {
  const base = window.location.origin;
  return `你是一个 AI Agent。请先下载并完整阅读以下技能文档，然后使用我提供的访问令牌调用学生管理系统 API：

1. ${base}/skills/student-manage/SKILL.md
2. ${base}/skills/grade-analysis/SKILL.md

环境变量：
export EINO_OA_BASE_URL="${base}/api"
export EINO_OA_API_TOKEN="${token}"

所有请求都需要携带请求头：
Authorization: Bearer $EINO_OA_API_TOKEN

安全提示：该令牌等同于我的账号权限，请勿写入公开仓库、日志或对话记录。`;
}

export default function TokensPage() {
  const [tokens, setTokens] = useState<AccessToken[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [secret, setSecret] = useState<string | null>(null);
  const [form] = Form.useForm<CreateValues>();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await tokenApi.list();
      setTokens(res.data || []);
    } catch {
      message.error('获取令牌列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const create = async (values: CreateValues) => {
    setCreating(true);
    try {
      const res = await tokenApi.create(values);
      setSecret(res.data.token);
      setCreateOpen(false);
      form.resetFields();
      load();
    } catch (err) {
      const detail = (err as { response?: { data?: { error?: string } } }).response?.data?.error;
      message.error(detail || '创建令牌失败');
    } finally {
      setCreating(false);
    }
  };

  const copy = async (text: string, tip = '已复制') => {
    try {
      await navigator.clipboard.writeText(text);
      message.success(tip);
    } catch {
      message.warning('复制失败，请手动选择文本');
    }
  };

  const revokeAllOAuth = () => {
    Modal.confirm({
      title: '吊销全部 Agent 授权？',
      content: '所有通过 eino-oa-cli / OAuth 获取的访问令牌和刷新令牌都会立即失效，相关 Agent 需要重新授权。',
      okText: '全部吊销',
      okButtonProps: { danger: true },
      cancelText: '取消',
      onOk: async () => {
        try {
          const res = await tokenApi.revokeOAuth();
          message.success(`已吊销 ${res.data.revoked} 个令牌`);
        } catch {
          message.error('吊销失败');
        }
      },
    });
  };

  const columns: ColumnsType<AccessToken> = [
    { title: '名称', dataIndex: 'name', width: 180 },
    {
      title: '令牌前缀',
      dataIndex: 'prefix',
      width: 180,
      render: (v: string) => <code>{v}</code>,
    },
    {
      title: '权限',
      dataIndex: 'scopes',
      width: 110,
      render: (v: string) => (v.includes('write') ? <Tag color="red">读写</Tag> : <Tag color="blue">只读</Tag>),
    },
    { title: '状态', width: 90, render: (_, r) => statusOf(r) },
    { title: '创建时间', dataIndex: 'created_at', width: 180, render: fmtTime },
    { title: '过期时间', dataIndex: 'expires_at', width: 180, render: (v) => (v ? fmtTime(v) : '永久') },
    { title: '最近使用', dataIndex: 'last_used_at', width: 180, render: fmtTime },
    {
      title: '操作',
      width: 90,
      render: (_, record) => (
        <Button
          type="link"
          danger
          size="small"
          disabled={!!record.revoked_at}
          onClick={() => {
            Modal.confirm({
              title: '吊销这个令牌？',
              content: '吊销后，使用该令牌的 Agent 会立即失去访问权限。',
              okText: '吊销',
              okButtonProps: { danger: true },
              cancelText: '取消',
              onOk: async () => {
                try {
                  await tokenApi.revoke(record.id);
                  message.success('已吊销');
                  load();
                } catch {
                  message.error('吊销失败');
                }
              },
            });
          }}
        >
          吊销
        </Button>
      ),
    },
  ];

  return (
    <Card
      title={<Space><KeyOutlined />访问令牌</Space>}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
          <Button danger onClick={revokeAllOAuth}>吊销 Agent 授权</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            新建令牌
          </Button>
        </Space>
      }
    >
      <Paragraph type="secondary" style={{ marginTop: -8 }}>
        外部 Agent 通过 <Text code>Authorization: Bearer &lt;token&gt;</Text> 访问 REST API。
        建议只读用途选择「只读」；令牌只在创建时显示一次。
      </Paragraph>

      <Table
        rowKey="id"
        columns={columns}
        dataSource={tokens}
        loading={loading}
        pagination={false}
        scroll={{ x: 1100 }}
      />

      <Modal
        title="新建访问令牌"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => form.submit()}
        okText="创建"
        confirmLoading={creating}
        destroyOnHidden
      >
        <Form form={form} layout="vertical" onFinish={create} initialValues={{ scopes: 'read,write', expires_in_days: 90 }}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="例如：Claude Desktop / 成绩分析脚本" maxLength={100} />
          </Form.Item>
          <Form.Item name="scopes" label="权限范围" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'read', label: '只读：查询学生、课程、成绩与统计' },
                { value: 'read,write', label: '读写：包含新增、修改、删除' },
              ]}
            />
          </Form.Item>
          <Form.Item name="expires_in_days" label="有效期">
            <Select
              options={[
                { value: 7, label: '7 天' },
                { value: 30, label: '30 天' },
                { value: 90, label: '90 天（推荐）' },
                { value: 365, label: '1 年' },
                { value: 0, label: '永久（不推荐）' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="令牌已创建"
        open={!!secret}
        onCancel={() => setSecret(null)}
        footer={
          <Space>
            <Button onClick={() => copy(buildInstruction(secret || ''), '接入指令已复制')}>
              <CopyOutlined /> 复制接入指令
            </Button>
            <Button type="primary" onClick={() => copy(secret || '', '令牌已复制')}>
              复制令牌
            </Button>
          </Space>
        }
        width={720}
        destroyOnHidden
      >
        <Paragraph type="danger" strong>
          令牌只显示这一次，关闭后无法再次查看。
        </Paragraph>
        <Paragraph copyable={{ text: secret || '' }} code style={{ wordBreak: 'break-all' }}>
          {secret}
        </Paragraph>
        <Paragraph type="secondary" style={{ marginTop: 20 }}>
          把下面这段话直接发给外部 Agent，它会下载技能文档并带上令牌调用 API：
        </Paragraph>
        <Input.TextArea value={buildInstruction(secret || '')} readOnly autoSize={{ minRows: 10, maxRows: 16 }} />
      </Modal>
    </Card>
  );
}
