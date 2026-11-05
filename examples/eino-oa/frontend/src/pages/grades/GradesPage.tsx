import { useState, useEffect, useCallback } from 'react';
import {
  Table, Button, Space, Input, Select, Modal, Form, message, Card, Popconfirm, Row, Col,
  InputNumber, Statistic, Tag,
} from 'antd';
import { PlusOutlined, SearchOutlined, ReloadOutlined, BarChartOutlined, WarningOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { gradeApi, studentApi, courseApi, type Grade, type Student, type Course, type GradeStats, type StudentGPA } from '../../services/api';

export default function GradesPage() {
  const [data, setData] = useState<Grade[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [studentId, setStudentId] = useState<string>('');
  const [courseId, setCourseId] = useState<string>('');
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Grade | null>(null);
  const [form] = Form.useForm();

  const [students, setStudents] = useState<Student[]>([]);
  const [courses, setCourses] = useState<Course[]>([]);

  const [statsOpen, setStatsOpen] = useState(false);
  const [stats, setStats] = useState<GradeStats | null>(null);
  const [warningOpen, setWarningOpen] = useState(false);
  const [warningData, setWarningData] = useState<StudentGPA[]>([]);

  useEffect(() => {
    studentApi.list({ page_size: 1000 }).then(res => setStudents(res.data.items || []));
    courseApi.list({ page_size: 1000 }).then(res => setCourses(res.data.items || []));
  }, []);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, unknown> = { page, page_size: pageSize };
      if (studentId) params.student_id = studentId;
      if (courseId) params.course_id = courseId;
      const res = await gradeApi.list(params);
      setData(res.data.items || []);
      setTotal(res.data.total);
    } catch {
      message.error('获取成绩列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, studentId, courseId]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleSave = async (values: Grade) => {
    try {
      if (editing?.id) {
        await gradeApi.update(editing.id, values);
        message.success('更新成功');
      } else {
        await gradeApi.create(values);
        message.success('录入成功');
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
      await gradeApi.delete(id);
      message.success('删除成功');
      fetchData();
    } catch {
      message.error('删除失败');
    }
  };

  const openEdit = (record: Grade) => {
    setEditing(record);
    form.setFieldsValue(record);
    setModalOpen(true);
  };

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ status: '正常' });
    setModalOpen(true);
  };

  const showStats = async (cid: number) => {
    try {
      const res = await gradeApi.statsByCourse(cid);
      setStats(res.data);
      setStatsOpen(true);
    } catch {
      message.error('获取统计失败');
    }
  };

  const showWarning = async () => {
    try {
      const res = await gradeApi.academicWarning(2.0);
      setWarningData(res.data.students || []);
      setWarningOpen(true);
    } catch {
      message.error('获取预警信息失败');
    }
  };

  const scoreColor = (score: number) => {
    if (score >= 90) return 'green';
    if (score >= 60) return 'blue';
    return 'red';
  };

  const columns: ColumnsType<Grade> = [
    { title: '学号', width: 120, render: (_, r) => r.student?.student_no },
    { title: '姓名', width: 100, render: (_, r) => r.student?.name },
    { title: '课程', width: 160, render: (_, r) => r.course?.name },
    {
      title: '成绩', dataIndex: 'score', width: 80,
      render: (v: number) => <Tag color={scoreColor(v)}>{v}</Tag>,
    },
    { title: '绩点', dataIndex: 'grade_point', width: 80 },
    { title: '学期', dataIndex: 'semester', width: 130 },
    {
      title: '状态', dataIndex: 'status', width: 80,
      render: (s: string) => <Tag color={s === '正常' ? 'green' : 'orange'}>{s}</Tag>,
    },
    {
      title: '操作', width: 200, fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button type="link" size="small" onClick={() => openEdit(record)}>编辑</Button>
          <Button type="link" size="small" onClick={() => showStats(record.course_id)} icon={<BarChartOutlined />}>统计</Button>
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
          <Select
            placeholder="选择学生" allowClear showSearch
            optionFilterProp="label"
            value={studentId || undefined}
            onChange={(v) => setStudentId(v || '')}
            style={{ width: 200 }}
            options={students.map(s => ({ value: String(s.id), label: `${s.student_no} - ${s.name}` }))}
          />
        </Col>
        <Col>
          <Select
            placeholder="选择课程" allowClear showSearch
            optionFilterProp="label"
            value={courseId || undefined}
            onChange={(v) => setCourseId(v || '')}
            style={{ width: 200 }}
            options={courses.map(c => ({ value: String(c.id), label: `${c.course_code} - ${c.name}` }))}
          />
        </Col>
        <Col>
          <Space>
            <Button icon={<SearchOutlined />} type="primary" onClick={fetchData}>搜索</Button>
            <Button icon={<ReloadOutlined />} onClick={() => { setStudentId(''); setCourseId(''); setPage(1); }}>重置</Button>
            <Button icon={<PlusOutlined />} type="primary" onClick={openCreate}>录入成绩</Button>
            <Button icon={<WarningOutlined />} danger onClick={showWarning}>学业预警</Button>
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
        title={editing ? '编辑成绩' : '录入成绩'} open={modalOpen}
        onCancel={() => { setModalOpen(false); setEditing(null); form.resetFields(); }}
        onOk={() => form.submit()} width={500} destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSave}>
          <Form.Item name="student_id" label="学生" rules={[{ required: true, message: '请选择学生' }]}>
            <Select
              showSearch optionFilterProp="label"
              options={students.map(s => ({ value: s.id, label: `${s.student_no} - ${s.name}` }))}
            />
          </Form.Item>
          <Form.Item name="course_id" label="课程" rules={[{ required: true, message: '请选择课程' }]}>
            <Select
              showSearch optionFilterProp="label"
              options={courses.map(c => ({ value: c.id, label: `${c.course_code} - ${c.name}` }))}
            />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="score" label="成绩" rules={[{ required: true, message: '请输入成绩' }]}>
                <InputNumber min={0} max={100} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="semester" label="学期">
                <Input placeholder="如：2025-2026-1" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="status" label="状态">
            <Select options={[
              { value: '正常', label: '正常' },
              { value: '补考', label: '补考' },
              { value: '重修', label: '重修' },
            ]} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="课程成绩统计" open={statsOpen} onCancel={() => setStatsOpen(false)} footer={null} width={500}>
        {stats && (
          <Row gutter={16}>
            <Col span={12}><Statistic title="课程" value={stats.course_name} /></Col>
            <Col span={12}><Statistic title="参考人数" value={stats.count} /></Col>
            <Col span={8}><Statistic title="平均分" value={stats.avg_score} precision={1} /></Col>
            <Col span={8}><Statistic title="最高分" value={stats.max_score} precision={1} /></Col>
            <Col span={8}><Statistic title="最低分" value={stats.min_score} precision={1} /></Col>
            <Col span={12}><Statistic title="及格率" value={stats.pass_rate} precision={1} suffix="%" /></Col>
          </Row>
        )}
      </Modal>

      <Modal title="学业预警 (GPA < 2.0)" open={warningOpen} onCancel={() => setWarningOpen(false)} footer={null} width={600}>
        <Table
          rowKey="student_id" dataSource={warningData} pagination={false} size="small"
          columns={[
            { title: '学号', dataIndex: 'student_no' },
            { title: '姓名', dataIndex: 'student_name' },
            { title: 'GPA', dataIndex: 'gpa', render: (v: number) => <Tag color="red">{v.toFixed(2)}</Tag> },
            { title: '课程数', dataIndex: 'total_courses' },
          ]}
        />
      </Modal>
    </Card>
  );
}
