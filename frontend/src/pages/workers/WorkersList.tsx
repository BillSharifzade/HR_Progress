import { useState } from 'react';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import {
  Alert, Avatar, Badge, Button, Col, DatePicker, Form, Input, InputNumber, Modal,
  Row, Select, Space, Table, Tag, Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined, SearchOutlined, UserOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';

import {
  listWorkers, createWorker, listSections,
  type CreateWorkerPayload,
} from '../../api/workers';
import { listDepartments, listGrades } from '../../api/competency';
import type { WorkerSummary } from '../../types';
import { PageSkeleton } from '../../components/PageSkeleton';
import { PageHeader } from '../../components/PageHeader';
import { useAuth } from '../../auth/useAuth';
import { OneFSyncButton } from './OneFSyncButton';

const GRADE_COLORS: Record<number, string> = {
  1: 'default', 2: 'geekblue', 3: 'cyan', 4: 'green', 5: 'orange',
};

function tenure(hiredAt: string): string {
  const years = dayjs().diff(dayjs(hiredAt), 'year');
  const months = dayjs().diff(dayjs(hiredAt).add(years, 'year'), 'month');
  const parts: string[] = [];
  if (years > 0) parts.push(`${years} г.`);
  if (months > 0) parts.push(`${months} мес.`);
  return parts.length ? parts.join(' ') : '< 1 мес.';
}

function CreateWorkerModal({ onClose }: { onClose: () => void }) {
  const [form] = Form.useForm();
  const qc = useQueryClient();
  const [selectedDeptId, setSelectedDeptId] = useState<string | undefined>();
  const [issued, setIssued] = useState<{ username: string; password: string } | null>(null);

  const { data: departments = [] } = useQuery({ queryKey: ['departments'], queryFn: listDepartments });
  const { data: grades = [] } = useQuery({ queryKey: ['grades'], queryFn: listGrades });
  const { data: sections = [] } = useQuery({
    queryKey: ['sections', selectedDeptId],
    queryFn: () => listSections(selectedDeptId),
    enabled: !!selectedDeptId,
  });

  const mut = useMutation({
    mutationFn: (v: CreateWorkerPayload) => createWorker(v),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['workers'] });
      setIssued({ username: res.username, password: res.password });
    },
  });

  const handleOk = async () => {
    const v = await form.validateFields();
    mut.mutate({
      full_name: v.full_name,
      email: v.email || null,
      birth_date: v.birth_date ? (v.birth_date as Dayjs).format('YYYY-MM-DD') : null,
      department_id: v.department_id || null,
      section_id: v.section_id || null,
      grade_id: v.grade_id || null,
      position: v.position || null,
      specialization: v.specialization || null,
      telegram_id: v.telegram_id || null,
      hired_at: v.hired_at ? (v.hired_at as Dayjs).format('YYYY-MM-DD') : null,
      hobbies: v.hobbies || null,
    });
  };

  if (issued) {
    return (
      <Modal
        open
        title="Сотрудник создан"
        onCancel={onClose}
        footer={<Button type="primary" onClick={onClose}>Готово</Button>}
        width={520}
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Alert
            type="warning"
            showIcon
            message="Передайте учётные данные сотруднику"
            description="Пароль больше не будет показан. Сотрудник не сможет сменить его самостоятельно — только администратор через раздел «Администрирование»."
          />
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Typography.Text type="secondary" style={{ minWidth: 72 }}>Логин:</Typography.Text>
            <Typography.Text code copyable={{ text: issued.username }}>{issued.username}</Typography.Text>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Typography.Text type="secondary" style={{ minWidth: 72 }}>Пароль:</Typography.Text>
            <Typography.Text code copyable={{ text: issued.password }}>{issued.password}</Typography.Text>
          </div>
        </Space>
      </Modal>
    );
  }

  return (
    <Modal
      open
      title="Новый сотрудник"
      onCancel={onClose}
      onOk={handleOk}
      okText="Создать"
      cancelText="Отмена"
      confirmLoading={mut.isPending}
      width={640}
      styles={{ body: { maxHeight: '72vh', overflowY: 'auto', paddingRight: 8 } }}
    >
      <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
        <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 12, fontWeight: 500 }}>
          Учётные данные
        </Typography.Text>
        <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 12, fontSize: 12 }}>
          Логин и временный пароль будут сгенерированы автоматически после создания.
        </Typography.Text>
        <Row gutter={16}>
          <Col span={24}>
            <Form.Item name="full_name" label="ФИО" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
          </Col>
        </Row>

        <Typography.Text type="secondary" style={{ display: 'block', margin: '8px 0 12px', fontWeight: 500 }}>
          Профиль
        </Typography.Text>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="email" label="Email">
              <Input type="email" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="birth_date" label="Дата рождения">
              <DatePicker format="DD.MM.YYYY" style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="hired_at" label="Дата приёма">
              <DatePicker format="DD.MM.YYYY" style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="department_id" label="Департамент">
              <Select allowClear placeholder="Выберите"
                options={departments.map((d) => ({ value: d.id, label: d.name }))}
                onChange={(v) => { setSelectedDeptId(v); form.setFieldValue('section_id', undefined); }}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="section_id" label="Отдел">
              <Select allowClear placeholder="Выберите" disabled={!selectedDeptId}
                options={sections.map((s) => ({ value: s.id, label: s.name }))}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="grade_id" label="Грейд">
              <Select allowClear placeholder="Выберите"
                options={grades.map((g) => ({ value: g.id, label: g.name }))}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="position" label="Должность">
              <Input placeholder="Введите должность" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="telegram_id" label="Telegram ID">
              <InputNumber style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={24}>
            <Form.Item name="hobbies" label="Хобби и увлечения">
              <Input.TextArea rows={2} />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}

export function WorkersList() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.roles.includes('HR_ADMIN') ?? false;
  const isSectionHead = user?.roles.includes('SECTION_HEAD') ?? false;
  const isDeptHead = user?.roles.includes('DEPT_HEAD') ?? false;

  // Forced scope for non-admins: SECTION_HEAD → their section; DEPT_HEAD → their dept.
  const forcedSectionId = !isAdmin && isSectionHead ? user?.scope_section_ids?.[0] : undefined;
  const forcedDeptId    = !isAdmin && !isSectionHead && isDeptHead ? user?.scope_department_ids?.[0] : undefined;

  const [search, setSearch] = useState('');
  const [deptId, setDeptId] = useState<string | undefined>(forcedDeptId);
  const [gradeId, setGradeId] = useState<string | undefined>();
  const [includeInactive, setIncludeInactive] = useState(false);
  const [createModal, setCreateModal] = useState(false);

  const effectiveDeptId    = forcedDeptId    ?? deptId;
  const effectiveSectionId = forcedSectionId;

  const { data: workers = [], isLoading } = useQuery({
    queryKey: ['workers', search, effectiveDeptId, effectiveSectionId, gradeId, includeInactive],
    queryFn: () => listWorkers({
      search: search || undefined,
      department_id: effectiveDeptId,
      section_id:    effectiveSectionId,
      grade_id: gradeId,
      include_inactive: includeInactive || undefined,
    }),
  });

  const { data: departments = [] } = useQuery({ queryKey: ['departments'], queryFn: listDepartments });
  const { data: grades = [] } = useQuery({ queryKey: ['grades'], queryFn: listGrades });

  const columns: ColumnsType<WorkerSummary> = [
    {
      title: 'Сотрудник',
      key: 'full_name',
      render: (_, r) => {
        const isSelf = user?.id === r.id;
        return (
          <Space>
            <Avatar size={32} icon={<UserOutlined />} style={{ flexShrink: 0 }} />
            <span>
              <Typography.Text strong style={{ display: 'block', lineHeight: 1.4 }}>
                {r.full_name}
                {isSelf && <Tag color="blue" style={{ marginInlineStart: 8 }}>это вы</Tag>}
              </Typography.Text>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                № {r.employee_no}
                {r.one_f_user_id != null && (
                  <Tag color="cyan" style={{ marginLeft: 6, fontSize: 11, lineHeight: '14px', padding: '0 4px' }}>
                    1F: {r.one_f_user_id}
                  </Tag>
                )}
              </Typography.Text>
            </span>
          </Space>
        );
      },
    },
    {
      title: 'Грейд', key: 'grade', width: 180,
      render: (_, r) => r.grade_name
        ? <Tag color={GRADE_COLORS[r.grade_level ?? 0] ?? 'default'}>{r.grade_name}</Tag>
        : <Typography.Text type="secondary">—</Typography.Text>,
    },
    {
      title: 'Должность', key: 'position', width: 200,
      render: (_, r) => r.position_name ?? <Typography.Text type="secondary">—</Typography.Text>,
    },
    {
      title: 'Департамент', key: 'department', width: 160,
      render: (_, r) => r.department_name ?? <Typography.Text type="secondary">—</Typography.Text>,
    },
    {
      title: 'Отдел', key: 'section', width: 200,
      render: (_, r) => r.section_name ?? <Typography.Text type="secondary">—</Typography.Text>,
    },
    {
      title: 'Стаж', key: 'tenure', width: 110,
      render: (_, r) => r.hired_at ? tenure(r.hired_at) : <Typography.Text type="secondary">—</Typography.Text>,
    },
    {
      title: 'Статус', key: 'status', width: 100,
      render: (_, r) => (
        <Badge status={r.is_active ? 'success' : 'default'} text={r.is_active ? 'Активен' : 'Неактивен'} />
      ),
    },
  ];

  if (isLoading) return <PageSkeleton type="list" />;

  return (
    <>
      <PageHeader
        title="Сотрудники"
        extra={
          isAdmin ? (
            <Space>
              <OneFSyncButton />
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModal(true)}>
                Добавить сотрудника
              </Button>
            </Space>
          ) : null
        }
      />
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Space wrap>
        <Input prefix={<SearchOutlined />} placeholder="Поиск: имя, ID, email, Telegram ID…"
          value={search} onChange={(e) => setSearch(e.target.value)} allowClear style={{ width: 300 }} />
        {isAdmin && (
          <Select placeholder="Департамент" allowClear style={{ width: 200 }} value={deptId} onChange={setDeptId}
            options={departments.map((d) => ({ value: d.id, label: d.name }))} />
        )}
        <Select placeholder="Грейд" allowClear style={{ width: 200 }} value={gradeId} onChange={setGradeId}
          options={grades.map((g) => ({ value: g.id, label: g.name }))} />
        <Button type={includeInactive ? 'primary' : 'default'} onClick={() => setIncludeInactive((v) => !v)}>
          {includeInactive ? 'Все сотрудники' : 'Только активные'}
        </Button>
      </Space>

      <Table
        rowKey="id"
        columns={columns}
        dataSource={workers}
        pagination={{ pageSize: 20, showSizeChanger: false }}
        onRow={(r) => ({ onClick: () => navigate(`/workers/${r.id}`) })}
        style={{ cursor: 'pointer' }}
        size="middle"
        locale={{ emptyText: 'Сотрудники не найдены' }}
      />

        {createModal && <CreateWorkerModal onClose={() => setCreateModal(false)} />}
      </Space>
    </>
  );
}
