import { useState, useEffect, useCallback } from 'react';
import {
  Table, Button, Space, Input, Select, Modal, Form, message, Card, Tag, Popconfirm, Row, Col,
} from 'antd';
import { PlusOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { studentApi, type Student } from '../../services/api';

const statusColors: Record<string, string> = {
  '在读': 'green',
  '休学': 'orange',
  '毕业': 'blue',
  '退学': 'red',
};

export default function StudentsPage() {
  const [data, setData] = useState<Student[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState({ name: '', student_no: '', department: '', status: '' });
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Student | null>(null);
  const [form] = Form.useForm();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await studentApi.list({ ...search, page, page_size: pageSize });
      setData(res.data.items || []);
      setTotal(res.data.total);
    } catch {
      message.error('获取学生列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleSave = async (values: Student) => {
    try {
      if (editing?.id) {
        await studentApi.update(editing.id, values);
        message.success('更新成功');
      } else {
        await studentApi.create(values);
        message.success('创建成功');
      }
      setModalOpen(false);
      form.resetFields();
      setEditing(null);
      fetchData();
    } catch {
      message.error('保存失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await studentApi.delete(id);
      message.success('删除成功');
      fetchData();
    } catch {
      message.error('删除失败');
    }
  };

  const openEdit = (record: Student) => {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  };

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ status: '在读', enrollment_year: new Date().getFullYear() });
    setModalOpen(true);
  };

  const columns: ColumnsType<Student> = [
    { title: '学号', dataIndex: 'student_no', width: 120 },
    { title: '姓名', dataIndex: 'name', width: 100 },
    { title: '性别', dataIndex: 'gender', width: 70 },
    { title: '年龄', dataIndex: 'age', width: 70 },
    { title: '院系', dataIndex: 'department', width: 140 },
    { title: '专业', dataIndex: 'major', width: 140 },
    { title: '班级', dataIndex: 'class_name', width: 100 },
    { title: '入学年份', dataIndex: 'enrollment_year', width: 100 },
    {
      title: '状态', dataIndex: 'status', width: 80,
      render: (s: string) => <Tag color={statusColors[s] || 'default'}>{s}</Tag>,
    },
    {
      title: '操作', width: 150, fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button type="link" size="small" onClick={() => openEdit(record)}>编辑</Button>
          <Popconfirm title="确定删除？" onConfirm={() => handleDelete(record.id!)}>
            <Button type="link" size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col>
          <Input
            placeholder="学号" prefix={<SearchOutlined />} allowClear
            value={search.student_no}
            onChange={(e) => setSearch(s => ({ ...s, student_no: e.target.value }))}
            onPressEnter={fetchData} style={{ width: 150 }}
          />
        </Col>
        <Col>
          <Input
            placeholder="姓名" allowClear value={search.name}
            onChange={(e) => setSearch(s => ({ ...s, name: e.target.value }))}
            onPressEnter={fetchData} style={{ width: 120 }}
          />
        </Col>
        <Col>
          <Input
            placeholder="院系" allowClear value={search.department}
            onChange={(e) => setSearch(s => ({ ...s, department: e.target.value }))}
            onPressEnter={fetchData} style={{ width: 140 }}
          />
        </Col>
        <Col>
          <Select
            placeholder="状态" allowClear value={search.status || undefined}
            onChange={(v) => setSearch(s => ({ ...s, status: v || '' }))}
            style={{ width: 100 }}
            options={[
              { value: '在读', label: '在读' },
              { value: '休学', label: '休学' },
              { value: '毕业', label: '毕业' },
              { value: '退学', label: '退学' },
            ]}
          />
        </Col>
        <Col>
          <Space>
            <Button icon={<SearchOutlined />} type="primary" onClick={fetchData}>搜索</Button>
            <Button icon={<ReloadOutlined />} onClick={() => { setSearch({ name: '', student_no: '', department: '', status: '' }); setPage(1); }}>重置</Button>
            <Button icon={<PlusOutlined />} type="primary" onClick={openCreate}>新增学生</Button>
          </Space>
        </Col>
      </Row>

      <Table
        rowKey="id" columns={columns} dataSource={data} loading={loading}
        scroll={{ x: 1200 }}
        pagination={{
          current: page, pageSize, total, showSizeChanger: true, showTotal: (t) => `共 ${t} 条`,
          onChange: (p, ps) => { setPage(p); setPageSize(ps); },
        }}
      />

      <Modal
        title={editing ? '编辑学生' : '新增学生'} open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditing(null); form.resetFields(); }}
        onOk={() => form.submit()} width={640} destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSave}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="student_no" label="学号" rules={[{ required: true, message: '请输入学号' }]}>
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="name" label="姓名" rules={[{ required: true, message: '请输入姓名' }]}>
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="gender" label="性别">
                <Select options={[{ value: '男', label: '男' }, { value: '女', label: '女' }]} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="age" label="年龄">
                <Input type="number" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="enrollment_year" label="入学年份">
                <Input type="number" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="department" label="院系">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="major" label="专业">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="class_name" label="班级">
                <Input />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="phone" label="电话">
                <Input />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="email" label="邮箱">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="status" label="状态">
            <Select options={[
              { value: '在读', label: '在读' },
              { value: '休学', label: '休学' },
              { value: '毕业', label: '毕业' },
              { value: '退学', label: '退学' },
            ]} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
