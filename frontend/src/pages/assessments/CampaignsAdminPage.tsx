import { useState } from 'react';
import {
  Button, Card, DatePicker, Form, Input, InputNumber, Modal, Select, Space, Table, Tag, message,
} from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';

import { PageHeader } from '../../components/PageHeader';
import { PageSkeleton } from '../../components/PageSkeleton';
import {
  listPeriods, createPeriod, listAllDepartments, listCompetencies,
} from '../../api/competency';
import { listSections } from '../../api/workers';
import type { AssessmentPeriod, CampaignStatus } from '../../types';
import { CampaignStatusColor, CampaignStatusLabel } from '../../types';

export function CampaignsAdminPage() {
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [msg, ctx] = message.useMessage();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm();

  const { data: periods = [], isLoading } = useQuery({
    queryKey: ['admin-periods'],
    queryFn: () => listPeriods(),
  });
  const { data: departments = [] } = useQuery({ queryKey: ['all-departments'], queryFn: listAllDepartments });
  const { data: competencies = [] } = useQuery({ queryKey: ['competencies'], queryFn: listCompetencies });
  const { data: sections = [] } = useQuery({ queryKey: ['sections'], queryFn: () => listSections() });

  const createMut = useMutation({
    mutationFn: createPeriod,
    onSuccess: (p) => {
      msg.success('Кампания создана');
      qc.invalidateQueries({ queryKey: ['admin-periods'] });
      setOpen(false);
      form.resetFields();
      navigate(`/admin/assessments/${p.id}`);
    },
    onError: () => msg.error('Не удалось создать кампанию'),
  });

  const handleCreate = async () => {
    const v = await form.validateFields();
    createMut.mutate({
      title: v.title,
      department_id: v.department_ids?.[0],
      department_ids: v.department_ids ?? [],
      section_ids: v.section_ids ?? [],
      period_start: v.range[0].format('YYYY-MM-DD'),
      period_end: v.range[1].format('YYYY-MM-DD'),
      group_size: v.group_size ?? 12,
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
        `${dayjs(r.period_start).format('DD.MM.YYYY')} — ${dayjs(r.period_end).format('DD.MM.YYYY')}`,
    },
    { title: 'Размер группы', dataIndex: 'group_size', key: 'group_size', width: 130 },
    {
      title: '', key: 'actions', width: 120,
      render: (_: unknown, r: AssessmentPeriod) => (
        <Button type="link" onClick={() => navigate(`/admin/assessments/${r.id}`)}>Управление</Button>
      ),
    },
  ];

  return (
    <>
      {ctx}
      <PageHeader
        title="Кампании ассессмента"
        subtitle="Создание и управление кампаниями оценки"
        extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>Новая кампания</Button>}
      />
      <Card>
        <Table rowKey="id" columns={columns} dataSource={periods} pagination={{ pageSize: 15 }} />
      </Card>

      <Modal
        title="Новая кампания ассессмента"
        open={open}
        onCancel={() => setOpen(false)}
        onOk={handleCreate}
        confirmLoading={createMut.isPending}
        okText="Создать"
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
          <Form.Item name="criteria" label="Критерии оценки (компетенции)" rules={[{ required: true, message: 'Выберите хотя бы одну' }]}>
            <Select
              mode="multiple" allowClear placeholder="Выберите компетенции"
              options={competencies.map(c => ({ value: c.id, label: `${c.name} (${c.kind})` }))}
              optionFilterProp="label"
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
