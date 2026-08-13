import { useState } from 'react';
import dayjs, { type Dayjs } from 'dayjs';
import {
  Button, DatePicker, Drawer, Empty, Form, Input, Popconfirm, Space, Table, Tooltip, Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, EditOutlined, InfoCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  createExperience, deleteExperience, listExperience, updateExperience,
  type ExperiencePayload,
} from '../../../api/workers';
import type { WorkerExperience } from '../../../types';

function fmtPeriod(r: WorkerExperience) {
  const from = r.started_on ? dayjs(r.started_on).format('MM.YYYY') : null;
  const to = r.ended_on ? dayjs(r.ended_on).format('MM.YYYY') : null;
  if (!from && !to) return '—';
  return `${from ?? '…'} — ${to ?? 'по н.в.'}`;
}

function ExperienceDrawer({
  workerId, entry, onClose,
}: { workerId: string; entry: WorkerExperience | null; onClose: () => void }) {
  const [form] = Form.useForm();
  const qc = useQueryClient();
  const isEdit = !!entry;

  const mut = useMutation({
    mutationFn: (v: ExperiencePayload) =>
      isEdit ? updateExperience(workerId, entry!.id, v) : createExperience(workerId, v),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['worker-experience', workerId] });
      onClose();
    },
  });

  const submit = async () => {
    const v = await form.validateFields();
    mut.mutate({
      company: v.company,
      position: v.position || null,
      started_on: v.started_on ? (v.started_on as Dayjs).format('YYYY-MM-DD') : null,
      ended_on: v.ended_on ? (v.ended_on as Dayjs).format('YYYY-MM-DD') : null,
      description: v.description || null,
    });
  };

  return (
    <Drawer
      open width={460} onClose={onClose}
      title={isEdit ? 'Изменить место работы' : 'Добавить место работы'}
      footer={
        <Space style={{ justifyContent: 'flex-end', width: '100%' }}>
          <Button onClick={onClose}>Отмена</Button>
          <Button type="primary" onClick={submit} loading={mut.isPending}>
            {isEdit ? 'Сохранить' : 'Добавить'}
          </Button>
        </Space>
      }
    >
      <Form
        form={form} layout="vertical"
        initialValues={entry ? {
          company: entry.company,
          position: entry.position,
          started_on: entry.started_on ? dayjs(entry.started_on) : null,
          ended_on: entry.ended_on ? dayjs(entry.ended_on) : null,
          description: entry.description,
        } : undefined}
      >
        <Form.Item name="company" label="Компания" rules={[{ required: true, message: 'Укажите компанию' }]}>
          <Input placeholder='ЗАО "Пример"' />
        </Form.Item>
        <Form.Item name="position" label="Должность">
          <Input placeholder="Ведущий специалист" />
        </Form.Item>
        <Space size="middle" style={{ display: 'flex' }}>
          <Form.Item name="started_on" label="Начало" style={{ flex: 1 }}>
            <DatePicker picker="month" format="MM.YYYY" style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item
            name="ended_on" label="Окончание" style={{ flex: 1 }}
            dependencies={['started_on']}
            rules={[({ getFieldValue }) => ({
              validator(_, value: Dayjs | null) {
                const start = getFieldValue('started_on') as Dayjs | null;
                if (!value || !start || !value.isBefore(start)) return Promise.resolve();
                return Promise.reject(new Error('Раньше начала'));
              },
            })]}
          >
            <DatePicker picker="month" format="MM.YYYY" style={{ width: '100%' }} placeholder="по наст. время" />
          </Form.Item>
        </Space>
        <Form.Item name="description" label="Примечание">
          <Input.TextArea rows={3} placeholder="Обязанности, достижения…" />
        </Form.Item>
        {entry?.raw_text && (
          <Typography.Paragraph type="secondary" style={{ fontSize: 12, marginBottom: 0 }}>
            Исходный текст анкеты: «{entry.raw_text}»
          </Typography.Paragraph>
        )}
      </Form>
    </Drawer>
  );
}

/**
 * «Опыт работы» — employment before the company, from the questionnaire's
 * free-text answer. Rows split out by the importer keep their original
 * fragment in `raw_text`, so a wrong split is visible and fixable.
 */
export function ExperienceTab({ workerId, canEdit }: { workerId: string; canEdit: boolean }) {
  const qc = useQueryClient();
  const [drawer, setDrawer] = useState<{ open: boolean; entry: WorkerExperience | null }>({
    open: false, entry: null,
  });

  const { data: rows = [] } = useQuery({
    queryKey: ['worker-experience', workerId],
    queryFn: () => listExperience(workerId),
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteExperience(workerId, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['worker-experience', workerId] }),
  });

  const columns: ColumnsType<WorkerExperience> = [
    {
      title: 'Компания', dataIndex: 'company', key: 'company',
      render: (v: string, r) => (
        <Space size={6}>
          <span style={{ fontWeight: 500 }}>{v}</span>
          {r.source === 'form' && r.raw_text && (
            <Tooltip title={`Из анкеты: «${r.raw_text}»`}>
              <InfoCircleOutlined style={{ opacity: 0.45 }} />
            </Tooltip>
          )}
        </Space>
      ),
    },
    { title: 'Должность', dataIndex: 'position', key: 'position', render: (v) => v ?? '—' },
    { title: 'Период', key: 'period', width: 170, render: (_, r) => fmtPeriod(r) },
    { title: 'Примечание', dataIndex: 'description', key: 'description', render: (v) => v ?? '—' },
    ...(canEdit ? [{
      key: 'actions', width: 88,
      render: (_: unknown, r: WorkerExperience) => (
        <Space size={0}>
          <Button type="text" size="small" icon={<EditOutlined />}
            onClick={() => setDrawer({ open: true, entry: r })} />
          <Popconfirm title="Удалить запись?" okText="Да" cancelText="Нет"
            onConfirm={() => remove.mutate(r.id)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    }] : []),
  ];

  return (
    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
      {canEdit && (
        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <Button icon={<PlusOutlined />} onClick={() => setDrawer({ open: true, entry: null })}>
            Добавить место работы
          </Button>
        </div>
      )}
      <Table
        rowKey="id" size="small" columns={columns} dataSource={rows} pagination={false}
        locale={{ emptyText: <Empty description="Опыт работы не указан" /> }}
      />
      {drawer.open && (
        <ExperienceDrawer
          workerId={workerId} entry={drawer.entry}
          onClose={() => setDrawer({ open: false, entry: null })}
        />
      )}
    </Space>
  );
}
