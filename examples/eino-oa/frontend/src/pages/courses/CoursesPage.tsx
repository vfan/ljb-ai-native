import { useState, useEffect, useCallback } from 'react';
import {
  Table, Button, Space, Input, Modal, Form, message, Card, Popconfirm, Row, Col, InputNumber,
} from 'antd';
import { PlusOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { courseApi, type Course } from '../../services/api';

export default function CoursesPage() {
  const [data, setData] = useState<Course[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [search, setSearch] = useState({ name: '', course_code: '', teacher: '' });
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Course | null>(null);
  const [form] = Form.useForm();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await courseApi.list({ ...search, page, page_size: pageSize });
      setData(res.data.items || []);
      setTotal(res.data.total);
    } catch {
      message.error('获取课程列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, search]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleSave = async (values: Course) => {
    try {
      if (editing?.id) {
        await courseApi.update(editing.id, values);
        message.success('更新成功');
      } else {
        await courseApi.create(values);
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
      await courseApi.delete(id);
      message.success('删除成功');
      fetchData();
    } catch {
      message.error('删除失败');
    }
  };

  const openEdit = (record: Course) => {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  };

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const columns: ColumnsType<Course> = [
    { title: '课程编号', dataIndex: 'course_code', width: 120 },
    { title: '课程名称', dataIndex: 'name', width: 160 },
    { title: '学分', dataIndex: 'credits', width: 80 },
    { title: '开课院系', dataIndex: 'department', width: 140 },
    { title: '任课教师', dataIndex: 'teacher', width: 100 },
    { title: '学期', dataIndex: 'semester', width: 130 },
    { title: '描述', dataIndex: 'description', ellipsis: true },
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
            placeholder="课程编号" prefix={<SearchOutlined />} allowClear
            value={search.course_code}
            onChange={(e) => setSearch(s => ({ ...s, course_code: e.target.value }))}
            onPressEnter={fetchData} style={{ width: 150 }}
          />
        </Col>
        <Col>
          <Input
            placeholder="课程名称" allowClear value={search.name}
            onChange={(e) => setSearch(s => ({ ...s, name: e.target.value }))}
            onPressEnter={fetchData} style={{ width: 150 }}
          />
        </Col>
        <Col>
          <Input
            placeholder="教师" allowClear value={search.teacher}
            onChange={(e) => setSearch(s => ({ ...s, teacher: e.target.value }))}
            onPressEnter={fetchData} style={{ width: 120 }}
          />
        </Col>
        <Col>
          <Space>
            <Button icon={<SearchOutlined />} type="primary" onClick={fetchData}>搜索</Button>
            <Button icon={<ReloadOutlined />} onClick={() => { setSearch({ name: '', course_code: '', teacher: '' }); setPage(1); }}>重置</Button>
            <Button icon={<PlusOutlined />} type="primary" onClick={openCreate}>新增课程</Button>
          </Space>
        </Col>
      </Row>

      <Table
        rowKey="id" columns={columns} dataSource={data} loading={loading}
        scroll={{ x: 1000 }}
        pagination={{
          current: page, pageSize, total, showSizeChanger: true, showTotal: (t) => `共 ${t} 条`,
          onChange: (p, ps) => { setPage(p); setPageSize(ps); },
        }}
      />

      <Modal
        title={editing ? '编辑课程' : '新增课程'} open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditing(null); form.resetFields(); }}
        onOk={() => form.submit()} width={600} destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSave}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="course_code" label="课程编号" rules={[{ required: true, message: '请输入课程编号' }]}>
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="name" label="课程名称" rules={[{ required: true, message: '请输入课程名称' }]}>
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="credits" label="学分">
                <InputNumber min={0} max={20} step={0.5} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="department" label="开课院系">
                <Input />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="teacher" label="任课教师">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="semester" label="学期">
            <Input placeholder="如：2025-2026-1" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
