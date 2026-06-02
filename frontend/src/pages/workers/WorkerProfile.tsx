import { useState } from 'react';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import {
  Avatar, Button, Card, DatePicker, Drawer, Empty,
  Form, Input, InputNumber, Popconfirm, Select, Space, Table, Tag, Tabs, Typography,
  theme as antdTheme,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckOutlined, CloseOutlined, DeleteOutlined, EditOutlined,
  PlusOutlined, SafetyCertificateOutlined, UserOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link, useParams } from 'react-router-dom';

import {
  getWorker, listCertifications, listHistory,
  createCertification, deleteCertification, createHistory,
  updateWorker, activateWorker, deactivateWorker,
  listSections,
  type CertificationPayload, type HistoryPayload, type UpdateWorkerPayload,
} from '../../api/workers';
import { listDepartments, listGrades } from '../../api/competency';
import type { WorkerCertification, WorkerHistory, UserRole } from '../../types';
import { UserRoleLabel } from '../../types';
import { useAuth } from '../../auth/useAuth';
import { canEditWorkers } from '../../auth/permissions';
import { PageSkeleton } from '../../components/PageSkeleton';
import { PageHeader } from '../../components/PageHeader';

const GRADE_COLORS: Record<number, string> = {
  1: 'default', 2: 'geekblue', 3: 'cyan', 4: 'green', 5: 'orange',
};
const EVENT_OPTIONS = [
  { value: 'HIRED', label: 'Принят' },
  { value: 'PROMOTED', label: 'Повышение' },
  { value: 'TRANSFERRED', label: 'Перевод' },
  { value: 'EXTERNAL_EXPERIENCE', label: 'Внешний опыт' },
  { value: 'COMMENT', label: 'Комментарий' },
  { value: 'OTHER', label: 'Другое' },
];
const EVENT_LABEL = Object.fromEntries(EVENT_OPTIONS.map(({ value, label }) => [value, label]));

function tenure(hiredAt: string) {
  const y = dayjs().diff(dayjs(hiredAt), 'year');
  const m = dayjs().diff(dayjs(hiredAt).add(y, 'year'), 'month');
  return [y > 0 && `${y} г.`, m > 0 && `${m} мес.`].filter(Boolean).join(' ') || '< 1 мес.';
}
function fmt(d?: string | null) { return d ? dayjs(d).format('DD.MM.YYYY') : '—'; }

// ─── Inline field row ────────────────────────────────────────────────────────
function Field({
  label, value, editing, control,
}: {
  label: string;
  value: React.ReactNode;
  editing: boolean;
  control: React.ReactNode;
}) {
  const { token } = antdTheme.useToken();
  return (
    <div style={{ display: 'flex', alignItems: 'center', padding: '10px 0', borderBottom: `1px solid ${token.colorBorderSecondary}` }}>
      <span style={{ width: 170, flexShrink: 0, fontSize: 13, color: token.colorTextSecondary, userSelect: 'none' }}>
        {label}
      </span>
      <div style={{ flex: 1, minWidth: 0 }}>
        {editing
          ? <div key="ctrl" className="field-control">{control}</div>
          : <span key="txt" className="field-text" style={{ color: value ? token.colorText : token.colorTextDisabled }}>{value ?? '—'}</span>
        }
      </div>
    </div>
  );
}

// ─── Add cert drawer (new entry — stays as drawer) ───────────────────────────
function AddCertDrawer({ workerId, onClose }: { workerId: string; onClose: () => void }) {
  const [form] = Form.useForm();
  const qc = useQueryClient();
  const mut = useMutation({
    mutationFn: (v: CertificationPayload) => createCertification(workerId, v),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['certifications', workerId] }); onClose(); },
  });
  const handleOk = async () => {
    const v = await form.validateFields();
    mut.mutate({
      title: v.title,
      issued_by: v.issued_by || null,
      issued_at: v.issued_at ? (v.issued_at as Dayjs).format('YYYY-MM-DD') : null,
      expires_at: v.expires_at ? (v.expires_at as Dayjs).format('YYYY-MM-DD') : null,
    });
  };
  return (
    <Drawer open title="Добавить сертификат" onClose={onClose} width={420}
      footer={
        <Space style={{ justifyContent: 'flex-end', width: '100%' }}>
          <Button onClick={onClose}>Отмена</Button>
          <Button type="primary" onClick={handleOk} loading={mut.isPending}>Добавить</Button>
        </Space>
      }>
      <Form form={form} layout="vertical">
        <Form.Item name="title" label="Название" rules={[{ required: true }]}>
          <Input placeholder="ACCA, IELTS, PMP…" />
        </Form.Item>
        <Form.Item name="issued_by" label="Организация"><Input /></Form.Item>
        <Form.Item name="issued_at" label="Дата получения">
          <DatePicker format="DD.MM.YYYY" style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="expires_at" label="Действует до">
          <DatePicker format="DD.MM.YYYY" style={{ width: '100%' }} />
        </Form.Item>
      </Form>
    </Drawer>
  );
}

// ─── Add history drawer (new entry — stays as drawer) ────────────────────────
function AddHistoryDrawer({ workerId, onClose }: { workerId: string; onClose: () => void }) {
  const [form] = Form.useForm();
  const qc = useQueryClient();
  const mut = useMutation({
    mutationFn: (v: HistoryPayload) => createHistory(workerId, v),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['history', workerId] }); onClose(); },
  });
  const handleOk = async () => {
    const v = await form.validateFields();
    mut.mutate({
      event_kind: v.event_kind,
      event_date: (v.event_date as Dayjs).format('YYYY-MM-DD'),
      title: v.title,
      description: v.description || null,
    });
  };
  return (
    <Drawer open title="Добавить запись" onClose={onClose} width={420}
      footer={
        <Space style={{ justifyContent: 'flex-end', width: '100%' }}>
          <Button onClick={onClose}>Отмена</Button>
          <Button type="primary" onClick={handleOk} loading={mut.isPending}>Добавить</Button>
        </Space>
      }>
      <Form form={form} layout="vertical" initialValues={{ event_kind: 'TRANSFERRED' }}>
        <Form.Item name="event_kind" label="Тип события" rules={[{ required: true }]}>
          <Select options={EVENT_OPTIONS} />
        </Form.Item>
        <Form.Item name="event_date" label="Дата" rules={[{ required: true }]}>
          <DatePicker format="DD.MM.YYYY" style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="title" label="Описание" rules={[{ required: true }]}>
          <Input.TextArea rows={3} />
        </Form.Item>
        <Form.Item name="description" label="Примечание">
          <Input.TextArea rows={2} />
        </Form.Item>
      </Form>
    </Drawer>
  );
}

// ─── Main ────────────────────────────────────────────────────────────────────
export function WorkerProfile() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const { user } = useAuth();
  const canEdit = canEditWorkers(user);
  const [form] = Form.useForm();
  const [isEditing, setIsEditing] = useState(false);
  const [editDeptId, setEditDeptId] = useState<string | undefined>();
  const [certDrawer, setCertDrawer] = useState(false);
  const [histDrawer, setHistDrawer] = useState(false);

  const { data: worker, isLoading } = useQuery({
    queryKey: ['worker', id], queryFn: () => getWorker(id!), enabled: !!id,
  });
  const { data: certifications = [] } = useQuery({
    queryKey: ['certifications', id], queryFn: () => listCertifications(id!), enabled: !!id,
  });
  const { data: history = [] } = useQuery({
    queryKey: ['history', id], queryFn: () => listHistory(id!), enabled: !!id,
  });
  const { data: departments = [] } = useQuery({ queryKey: ['departments'], queryFn: listDepartments });
  const { data: grades = [] } = useQuery({ queryKey: ['grades'], queryFn: listGrades });
  const { data: sections = [] } = useQuery({
    queryKey: ['sections', editDeptId],
    queryFn: () => listSections(editDeptId),
    enabled: isEditing && !!editDeptId,
  });

  const saveMut = useMutation({
    mutationFn: (payload: UpdateWorkerPayload) => updateWorker(id!, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['worker', id] });
      qc.invalidateQueries({ queryKey: ['workers'] });
      setIsEditing(false);
    },
  });

  const toggleActive = useMutation({
    mutationFn: () => (worker?.is_active ? deactivateWorker(id!) : activateWorker(id!)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['worker', id] });
      qc.invalidateQueries({ queryKey: ['workers'] });
    },
  });

  const deleteCert = useMutation({
    mutationFn: (certId: string) => deleteCertification(id!, certId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['certifications', id] }),
  });

  const startEdit = () => {
    if (!worker) return;
    form.setFieldsValue({
      full_name: worker.full_name,
      email: worker.email,
      personnel_number: worker.personnel_number,
      birth_date: worker.birth_date ? dayjs(worker.birth_date) : null,
      hired_at: worker.hired_at ? dayjs(worker.hired_at) : null,
      department_id: worker.department_id,
      section_id: worker.section_id,
      grade_id: worker.grade_id,
      position: worker.position,
      specialization: worker.specialization,
      telegram_id: worker.telegram_id,
      hobbies: worker.hobbies,
    });
    setEditDeptId(worker.department_id ?? undefined);
    setIsEditing(true);
  };

  const cancelEdit = () => { form.resetFields(); setIsEditing(false); };

  const handleSave = async () => {
    const v = await form.validateFields();
    saveMut.mutate({
      full_name: v.full_name,
      email: v.email || null,
      personnel_number: v.personnel_number || null,
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

  if (isLoading) return <PageSkeleton type="profile" />;
  if (!worker) return null;

  const certColumns: ColumnsType<WorkerCertification> = [
    { title: 'Сертификат', dataIndex: 'title', key: 'title' },
    { title: 'Организация', dataIndex: 'issued_by', key: 'issued_by', render: (v) => v ?? '—' },
    { title: 'Получен', key: 'a', width: 110, render: (_, r) => fmt(r.issued_at) },
    { title: 'До', key: 'b', width: 110, render: (_, r) => fmt(r.expires_at) },
    ...(canEdit ? [{
      key: 'del', width: 44,
      render: (_: unknown, r: WorkerCertification) => (
        <Popconfirm title="Удалить?" onConfirm={() => deleteCert.mutate(r.id)} okText="Да" cancelText="Нет">
          <Button type="text" danger icon={<DeleteOutlined />} size="small" />
        </Popconfirm>
      ),
    }] : []),
  ];

  const histColumns: ColumnsType<WorkerHistory> = [
    { title: 'Дата', dataIndex: 'event_date', key: 'date', width: 100, render: (v) => fmt(v) },
    { title: 'Тип', dataIndex: 'event_kind', key: 'kind', width: 120, render: (v) => EVENT_LABEL[v] ?? v },
    { title: 'Компания', key: 'c', width: 150, render: (_, r) => (r.meta as any)?.company_name ?? '—' },
    { title: 'Дпт', key: 'd', width: 100, render: (_, r) => (r.meta as any)?.department_name ?? '—' },
    { title: 'Отдел', key: 'e', width: 140, render: (_, r) => (r.meta as any)?.section_name ?? '—' },
    { title: 'Грейд', key: 'f', width: 140, render: (_, r) => (r.meta as any)?.grade_name ?? '—' },
    { title: 'Должность / описание', key: 'g', render: (_, r) => (r.meta as any)?.position_name ?? r.title },
  ];

  const subtitle = (
    <Space size={4} split={<span style={{ color: 'rgba(0,0,0,0.25)' }}>·</span>}>
      <Link to="/workers" style={{ color: 'inherit' }}>Сотрудники</Link>
      <span>{worker.full_name}</span>
    </Space>
  );

  const titleNode = isEditing ? (
    <Form.Item name="full_name" noStyle rules={[{ required: true }]}>
      <Input style={{ fontSize: 22, fontWeight: 600, width: 360 }} size="large" />
    </Form.Item>
  ) : (
    <Space align="center" wrap size={8}>
      <span>{worker.full_name}</span>
      <Tag color={worker.is_active ? 'success' : 'default'}>
        {worker.is_active ? 'Активен' : 'Неактивен'}
      </Tag>
      {worker.grade_name && (
        <Tag color={GRADE_COLORS[worker.grade_level ?? 0] ?? 'default'}>{worker.grade_name}</Tag>
      )}
    </Space>
  );

  const headerExtra = !canEdit ? null : isEditing ? (
    <Space className="edit-btn-group">
      <Button onClick={cancelEdit} icon={<CloseOutlined />}>Отмена</Button>
      <Button type="primary" icon={<CheckOutlined />} loading={saveMut.isPending} onClick={handleSave}>
        Сохранить
      </Button>
    </Space>
  ) : (
    <Space className="edit-btn-group">
      <Button icon={<EditOutlined />} onClick={startEdit}>Редактировать</Button>
      <Popconfirm
        title={worker.is_active ? 'Деактивировать сотрудника?' : 'Активировать сотрудника?'}
        onConfirm={() => toggleActive.mutate()}
        okText="Да" cancelText="Нет"
      >
        <Button danger={worker.is_active} type={worker.is_active ? 'default' : 'primary'} loading={toggleActive.isPending}>
          {worker.is_active ? 'Деактивировать' : 'Активировать'}
        </Button>
      </Popconfirm>
    </Space>
  );

  return (
    <Form form={form} component={false}>
      <PageHeader title={titleNode} subtitle={subtitle} extra={headerExtra} />
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* Profile summary card (no duplicate name; shows avatar + meta + roles) */}
        <Card size="small">
          <div style={{ display: 'flex', alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
            <Avatar size={64} icon={<UserOutlined />} style={{ flexShrink: 0 }} />
            <div style={{ flex: 1, minWidth: 0 }}>
              {!isEditing && (
                <Typography.Text type="secondary" style={{ display: 'block', fontSize: 13 }}>
                  {[worker.position ?? worker.position_name, worker.department_name, worker.section_name].filter(Boolean).join(' · ')}
                  {` · № ${worker.employee_no}`}
                  {worker.one_f_user_id != null && ` · 1F: ${worker.one_f_user_id}`}
                </Typography.Text>
              )}
              {!isEditing && (worker.roles ?? []).length > 0 && (
                <Space style={{ marginTop: 6 }} wrap>
                  {(worker.roles ?? []).map((r) => <Tag key={r} color="geekblue">{UserRoleLabel[r as UserRole] ?? r}</Tag>)}
                </Space>
              )}
            </div>
          </div>
        </Card>

        {/* Tabs */}
        <Tabs items={[
          {
            key: 'profile',
            label: 'Профиль',
            children: (
              <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                <Card title="Личные данные" size="small">
                  <Field label="ID сотрудника" value={`№ ${worker.employee_no}`} editing={false} control={null} />
                  {worker.one_f_user_id != null && (
                    <Field
                      label="ID в 1F"
                      value={
                        <span>
                          {worker.one_f_user_id}
                          {worker.last_synced_at && (
                            <Typography.Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
                              синхр.: {fmt(worker.last_synced_at)}
                            </Typography.Text>
                          )}
                        </span>
                      }
                      editing={false}
                      control={null}
                    />
                  )}
                  {worker.phone_number && (
                    <Field label="Телефон" value={worker.phone_number} editing={false} control={null} />
                  )}
                  <Field label="Дата рождения" value={fmt(worker.birth_date)} editing={isEditing}
                    control={<Form.Item name="birth_date" noStyle><DatePicker format="DD.MM.YYYY" size="small" /></Form.Item>} />
                  <Field label="Email" value={worker.email} editing={isEditing}
                    control={<Form.Item name="email" noStyle><Input type="email" size="small" style={{ maxWidth: 280 }} /></Form.Item>} />
                  <Field label="Дата приёма" value={fmt(worker.hired_at)} editing={isEditing}
                    control={<Form.Item name="hired_at" noStyle><DatePicker format="DD.MM.YYYY" size="small" /></Form.Item>} />
                  <Field
                    label="Стаж"
                    value={worker.hired_at ? tenure(worker.hired_at) : null}
                    editing={false}
                    control={null}
                  />
                  <Field label="Специализация" value={worker.specialization} editing={isEditing}
                    control={<Form.Item name="specialization" noStyle><Input size="small" style={{ maxWidth: 320 }} /></Form.Item>} />
                  <Field label="Telegram ID" value={worker.telegram_id} editing={isEditing}
                    control={<Form.Item name="telegram_id" noStyle><InputNumber size="small" style={{ width: 180 }} /></Form.Item>} />
                  <Field label="Хобби и увлечения" value={worker.hobbies} editing={isEditing}
                    control={
                      <Form.Item name="hobbies" noStyle>
                        <Input.TextArea rows={2} size="small" style={{ maxWidth: 420 }} />
                      </Form.Item>
                    } />
                </Card>

                <Card title="Организационная структура" size="small">
                  <Field label="Департамент" value={worker.department_name} editing={isEditing}
                    control={
                      <Form.Item name="department_id" noStyle>
                        <Select
                          allowClear size="small" style={{ width: 260 }}
                          options={departments.map((d) => ({ value: d.id, label: d.name }))}
                          onChange={(v) => {
                            setEditDeptId(v);
                            form.setFieldValue('section_id', undefined);
                          }}
                        />
                      </Form.Item>
                    } />
                  <Field label="Отдел" value={worker.section_name} editing={isEditing}
                    control={
                      <Form.Item name="section_id" noStyle>
                        <Select
                          allowClear size="small" style={{ width: 260 }}
                          disabled={!editDeptId}
                          options={sections.map((s) => ({ value: s.id, label: s.name }))}
                          placeholder={editDeptId ? 'Выберите отдел' : 'Сначала выберите департамент'}
                        />
                      </Form.Item>
                    } />
                  <Field label="Грейд" value={worker.grade_name} editing={isEditing}
                    control={
                      <Form.Item name="grade_id" noStyle>
                        <Select allowClear size="small" style={{ width: 260 }}
                          options={grades.map((g) => ({ value: g.id, label: g.name }))} />
                      </Form.Item>
                    } />
                  <Field label="Должность" value={worker.position ?? worker.position_name} editing={isEditing}
                    control={
                      <Form.Item name="position" noStyle>
                        <Input size="small" style={{ maxWidth: 260 }} placeholder="Введите должность" />
                      </Form.Item>
                    } />
                </Card>

                {!isEditing && (
                  <Space>
                    <Button disabled icon={<SafetyCertificateOutlined />}>Матрица компетенций</Button>
                    <Button disabled>ПИР</Button>
                  </Space>
                )}
              </Space>
            ),
          },
          {
            key: 'history',
            label: 'История перемещений',
            children: (
              <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                {canEdit && (
                  <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                    <Button icon={<PlusOutlined />} onClick={() => setHistDrawer(true)}>Добавить запись</Button>
                  </div>
                )}
                <Table rowKey="id" columns={histColumns} dataSource={history} size="small"
                  pagination={false} locale={{ emptyText: <Empty description="История пуста" /> }} />
              </Space>
            ),
          },
          {
            key: 'certifications',
            label: 'Сертификаты',
            children: (
              <Space direction="vertical" size="middle" style={{ width: '100%' }}>
                {canEdit && (
                  <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
                    <Button icon={<PlusOutlined />} onClick={() => setCertDrawer(true)}>Добавить сертификат</Button>
                  </div>
                )}
                <Table rowKey="id" columns={certColumns} dataSource={certifications} size="small"
                  pagination={false} locale={{ emptyText: <Empty description="Сертификаты не добавлены" /> }} />
              </Space>
            ),
          },
        ]} />

        {certDrawer && <AddCertDrawer workerId={id!} onClose={() => setCertDrawer(false)} />}
        {histDrawer && <AddHistoryDrawer workerId={id!} onClose={() => setHistDrawer(false)} />}
      </Space>
    </Form>
  );
}
