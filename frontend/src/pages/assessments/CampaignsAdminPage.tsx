import { useState } from 'react';
import {
  Button, Card, DatePicker, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag,
  Tooltip, message,
} from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';
import { formatPeriodRange } from '../../utils/period';

import { PageHeader } from '../../components/PageHeader';
import { PageSkeleton } from '../../components/PageSkeleton';
import {
  listPeriods, createPeriod, updatePeriod, deletePeriod, listAllDepartments, listCompetencies,
} from '../../api/competency';
import { listSections } from '../../api/workers';
import type { AssessmentPeriod, CampaignStatus } from '../../types';
import { CampaignStatusColor, CampaignStatusLabel } from '../../types';

export function CampaignsAdminPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [msg, ctx] = message.useMessage();
  const [open, setOpen] = useState(false);
  // null ⇒ the modal is creating; a period ⇒ it is editing that one.
  const [editing, setEditing] = useState<AssessmentPeriod | null>(null);
  const [form] = Form.useForm();

  const { data: periods = [], isLoading } = useQuery({
    queryKey: ['admin-periods'],
    queryFn: () => listPeriods(),
  });
  const { data: departments = [] } = useQuery({ queryKey: ['all-departments'], queryFn: listAllDepartments });
  const { data: competencies = [] } = useQuery({ queryKey: ['competencies'], queryFn: listCompetencies });
  const { data: sections = [] } = useQuery({ queryKey: ['sections'], queryFn: () => listSections() });

  const closeModal = () => {
    setOpen(false);
    setEditing(null);
    form.resetFields();
  };

  const createMut = useMutation({
    mutationFn: createPeriod,
    onSuccess: (p) => {
      msg.success('Кампания создана');
      qc.invalidateQueries({ queryKey: ['admin-periods'] });
      closeModal();
      navigate(`/admin/assessments/${p.id}`);
    },
    onError: () => msg.error('Не удалось создать кампанию'),
  });

  const updateMut = useMutation({
    mutationFn: (v: { id: string; payload: Parameters<typeof updatePeriod>[1] }) =>
      updatePeriod(v.id, v.payload),
    onSuccess: () => {
      msg.success('Кампания обновлена');
      qc.invalidateQueries({ queryKey: ['admin-periods'] });
      closeModal();
    },
    onError: (e: any) => msg.error(e?.response?.data?.error?.message ?? 'Не удалось сохранить кампанию'),
  });

  const deleteMut = useMutation({
    mutationFn: deletePeriod,
    onSuccess: () => {
      msg.success('Кампания удалена');
      qc.invalidateQueries({ queryKey: ['admin-periods'] });
    },
    onError: (e: any) => msg.error(e?.response?.data?.error?.message ?? 'Не удалось удалить кампанию'),
  });

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setOpen(true);
  };

  const openEdit = (p: AssessmentPeriod) => {
    setEditing(p);
    form.setFieldsValue({
      title: p.title,
      range: [dayjs(p.period_start), dayjs(p.period_end)],
      department_ids: p.department_ids ?? [],
      section_ids: p.section_ids ?? [],
      group_size: p.group_size,
    });
    setOpen(true);
  };

  const handleSubmit = async () => {
    const v = await form.validateFields();
    const base = {
      department_id: v.department_ids?.[0],
      department_ids: v.department_ids ?? [],
      section_ids: v.section_ids ?? [],
      period_start: v.range[0].format('YYYY-MM-DD'),
      period_end: v.range[1].format('YYYY-MM-DD'),
      group_size: v.group_size ?? 12,
      title: v.title,
    };
    if (editing) {
      updateMut.mutate({ id: editing.id, payload: base });
      return;
    }
    createMut.mutate({
      ...base,
      criteria: (v.criteria ?? []).map((cid: string) => ({ competency_id: cid })),
    });
  };

  if (isLoading) return <PageSkeleton type="list" />;

  const columns = [
    { title: 'Название', dataIndex: 'title', key: 'title' },
    {
      title: 'Статус', dataIndex: 'status', key: 'status',
      render: (s: CampaignStatus) => <Tag color={CampaignStatusColor[s]}>{CampaignStatusLabel[s]}</Tag>,
    },
    {
      title: 'Период', key: 'period',
      render: (_: unknown, r: AssessmentPeriod) =>
        formatPeriodRange(r.period_start, r.period_end),
    },
    { title: 'Размер группы', dataIndex: 'group_size', key: 'group_size', width: 130 },
    {
      title: '', key: 'actions', width: 260,
      render: (_: unknown, r: AssessmentPeriod) => (
        <Space size={0}>
          <Button type="link" onClick={() => navigate(`/admin/assessments/${r.id}`)}>Управление</Button>
          <Button type="link" onClick={() => openEdit(r)}>Изменить</Button>
          {r.status === 'published' ? (
            <Tooltip title="Опубликованную кампанию удалить нельзя">
              <Button type="link" danger disabled>Удалить</Button>
            </Tooltip>
          ) : (
            <Popconfirm
              title="Удалить кампанию?"
              description="Оценки, участники и группы этой кампании будут удалены безвозвратно."
              onConfirm={() => deleteMut.mutate(r.id)}
              okText="Удалить"
              okButtonProps={{ danger: true }}
              cancelText="Отмена"
            >
              <Button type="link" danger loading={deleteMut.isPending && deleteMut.variables === r.id}>
                Удалить
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <>
      {ctx}
      <PageHeader
        title="Кампании ассессмента"
        subtitle="Создание и управление кампаниями оценки"
        extra={<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>Новая кампания</Button>}
      />
      <Card>
        <Table rowKey="id" columns={columns} dataSource={periods} pagination={{ pageSize: 15 }} />
      </Card>

      <Modal
        title={editing ? 'Редактирование кампании' : 'Новая кампания ассессмента'}
        open={open}
        onCancel={closeModal}
        onOk={handleSubmit}
        confirmLoading={createMut.isPending || updateMut.isPending}
        okText={editing ? 'Сохранить' : 'Создать'}
        width={640}
      >
        <Form form={form} layout="vertical" initialValues={{ group_size: 12 }}>
          <Form.Item name="title" label="Название" rules={[{ required: true, message: 'Укажите название' }]}>
            <Input placeholder="Ассессмент Q3 2026" />
          </Form.Item>
          <Form.Item name="range" label="Период проведения" rules={[{ required: true, message: 'Укажите даты' }]}>
            <DatePicker.RangePicker format="DD.MM.YYYY" style={{ width: '100%' }} />
          </Form.Item>
          <Space style={{ display: 'flex' }} align="start">
            <Form.Item name="department_ids" label="Департаменты" style={{ flex: 1, minWidth: 260 }}>
              <Select
                mode="multiple" allowClear placeholder="Выберите департаменты"
                options={departments.map(d => ({ value: d.id, label: d.name }))}
                optionFilterProp="label"
              />
            </Form.Item>
            <Form.Item name="group_size" label="Размер группы">
              <InputNumber min={1} max={100} />
            </Form.Item>
          </Space>
          <Form.Item name="section_ids" label="Отделы (опционально)">
            <Select
              mode="multiple" allowClear placeholder="Выберите отделы"
              options={sections.map(s => ({ value: s.id, label: s.name }))}
              optionFilterProp="label"
            />
          </Form.Item>
          {/* Criteria are only seeded at creation; afterwards they are edited on
              the campaign page, which also carries the passing scores. */}
          {!editing && (
            <Form.Item name="criteria" label="Критерии оценки (компетенции)" rules={[{ required: true, message: 'Выберите хотя бы одну' }]}>
              <Select
                mode="multiple" allowClear placeholder="Выберите компетенции"
                options={competencies.map(c => ({ value: c.id, label: `${c.name} (${c.kind})` }))}
                optionFilterProp="label"
              />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </>
  );
}
