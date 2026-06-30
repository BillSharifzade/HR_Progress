import { useState } from 'react';
import {
  Button, Card, Drawer, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag,
  Timeline, Typography, message,
} from 'antd';
import { CopyOutlined, HistoryOutlined, PlusOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';

import { PageHeader } from '../../components/PageHeader';
import {
  listInterpretations, upsertInterpretation, deleteInterpretation, copyInterpretations,
  interpretationHistory, listAllDepartments, listGrades, listCompetencies,
} from '../../api/competency';
import type { Interpretation } from '../../types';

const { Text } = Typography;

export function InterpretationsPage() {
  const qc = useQueryClient();
  const [msg, ctx] = message.useMessage();
  const [filters, setFilters] = useState<{ department_id?: string; grade_id?: string; competency_id?: string }>({});
  const [editOpen, setEditOpen] = useState(false);
  const [copyOpen, setCopyOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [form] = Form.useForm();
  const [copyForm] = Form.useForm();

  const { data: departments = [] } = useQuery({ queryKey: ['all-departments'], queryFn: listAllDepartments });
  const { data: grades = [] } = useQuery({ queryKey: ['grades'], queryFn: listGrades });
  const { data: competencies = [] } = useQuery({ queryKey: ['competencies'], queryFn: listCompetencies });
  const { data: rows = [], isLoading } = useQuery({
    queryKey: ['interpretations', filters],
    queryFn: () => listInterpretations(filters),
  });
  const { data: history = [] } = useQuery({
    queryKey: ['interp-history', filters],
    queryFn: () => interpretationHistory(filters),
    enabled: historyOpen,
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ['interpretations'] });

  const saveMut = useMutation({
    mutationFn: upsertInterpretation,
    onSuccess: () => { msg.success('Сохранено'); setEditOpen(false); form.resetFields(); invalidate(); },
    onError: () => msg.error('Не удалось сохранить'),
  });
  const delMut = useMutation({
    mutationFn: deleteInterpretation,
    onSuccess: () => { msg.success('Удалено'); invalidate(); },
  });
  const copyMut = useMutation({
    mutationFn: copyInterpretations,
    onSuccess: (r) => { msg.success(`Скопировано записей: ${r.copied}`); setCopyOpen(false); copyForm.resetFields(); invalidate(); },
    onError: () => msg.error('Не удалось скопировать'),
  });

  const openEdit = (rec?: Interpretation) => {
    if (rec) {
      form.setFieldsValue({
        department_id: rec.department_id, grade_id: rec.grade_id,
        competency_id: rec.competency_id, score: rec.score, text: rec.text,
      });
    } else {
      form.resetFields();
    }
    setEditOpen(true);
  };

  return (
    <>
      {ctx}
      <PageHeader
        title="Справочник текстовых интерпретаций"
        subtitle="Интерпретации по связке: Департамент → Грейд → Компетенция → Балл"
        extra={
          <Space>
            <Button icon={<HistoryOutlined />} onClick={() => setHistoryOpen(true)}>История</Button>
            <Button icon={<CopyOutlined />} onClick={() => setCopyOpen(true)}>Копировать</Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openEdit()}>Добавить</Button>
          </Space>
        }
      />

      <Card>
        <Space style={{ marginBottom: 16 }} wrap>
          <Select allowClear placeholder="Департамент" style={{ width: 200 }}
            value={filters.department_id} onChange={(v) => setFilters(f => ({ ...f, department_id: v }))}
            options={departments.map(d => ({ value: d.id, label: d.name }))} optionFilterProp="label" showSearch />
          <Select allowClear placeholder="Грейд" style={{ width: 160 }}
            value={filters.grade_id} onChange={(v) => setFilters(f => ({ ...f, grade_id: v }))}
            options={grades.map(g => ({ value: g.id, label: g.name }))} optionFilterProp="label" showSearch />
          <Select allowClear placeholder="Компетенция" style={{ width: 240 }}
            value={filters.competency_id} onChange={(v) => setFilters(f => ({ ...f, competency_id: v }))}
            options={competencies.map(c => ({ value: c.id, label: c.name }))} optionFilterProp="label" showSearch />
        </Space>

        <Table rowKey="id" loading={isLoading} dataSource={rows} pagination={{ pageSize: 20 }}
          columns={[
            { title: 'Департамент', dataIndex: 'department_name', key: 'dept', width: 160 },
            { title: 'Грейд', dataIndex: 'grade_name', key: 'grade', width: 100 },
            { title: 'Компетенция', dataIndex: 'competency_name', key: 'comp', width: 180 },
            { title: 'Балл', dataIndex: 'score', key: 'score', width: 70, render: (s: number) => <Tag>{s}</Tag> },
            { title: 'Интерпретация', dataIndex: 'text', key: 'text' },
            { title: 'Версия', dataIndex: 'version', key: 'version', width: 80 },
            {
              title: '', key: 'actions', width: 140,
              render: (_: unknown, r: Interpretation) => (
                <Space>
                  <Button type="link" size="small" onClick={() => openEdit(r)}>Изменить</Button>
                  <Popconfirm title="Удалить?" onConfirm={() => delMut.mutate(r.id)} okText="Да" cancelText="Нет">
                    <Button type="link" danger size="small">Удалить</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>

      <Modal title="Текстовая интерпретация" open={editOpen} onCancel={() => setEditOpen(false)}
        onOk={async () => { const v = await form.validateFields(); saveMut.mutate(v); }}
        confirmLoading={saveMut.isPending} okText="Сохранить" width={560}>
        <Form form={form} layout="vertical">
          <Form.Item name="department_id" label="Департамент" rules={[{ required: true }]}>
            <Select options={departments.map(d => ({ value: d.id, label: d.name }))} optionFilterProp="label" showSearch />
          </Form.Item>
          <Form.Item name="grade_id" label="Грейд" rules={[{ required: true }]}>
            <Select options={grades.map(g => ({ value: g.id, label: g.name }))} optionFilterProp="label" showSearch />
          </Form.Item>
          <Form.Item name="competency_id" label="Компетенция" rules={[{ required: true }]}>
            <Select options={competencies.map(c => ({ value: c.id, label: c.name }))} optionFilterProp="label" showSearch />
          </Form.Item>
          <Form.Item name="score" label="Балл (1–10)" rules={[{ required: true }]}>
            <InputNumber min={1} max={10} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="text" label="Текст интерпретации" rules={[{ required: true }]}>
            <Input.TextArea autoSize={{ minRows: 3, maxRows: 8 }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="Копирование интерпретаций" open={copyOpen} onCancel={() => setCopyOpen(false)}
        onOk={async () => { const v = await copyForm.validateFields(); copyMut.mutate(v); }}
        confirmLoading={copyMut.isPending} okText="Копировать" width={560}>
        <Form form={copyForm} layout="vertical">
          <Form.Item name="from_department_id" label="Из департамента" rules={[{ required: true }]}>
            <Select options={departments.map(d => ({ value: d.id, label: d.name }))} optionFilterProp="label" showSearch />
          </Form.Item>
          <Form.Item name="to_department_id" label="В департамент" rules={[{ required: true }]}>
            <Select options={departments.map(d => ({ value: d.id, label: d.name }))} optionFilterProp="label" showSearch />
          </Form.Item>
          <Form.Item name="from_grade_id" label="Из грейда (опц.)">
            <Select allowClear options={grades.map(g => ({ value: g.id, label: g.name }))} optionFilterProp="label" showSearch />
          </Form.Item>
          <Form.Item name="to_grade_id" label="В грейд (опц.)">
            <Select allowClear options={grades.map(g => ({ value: g.id, label: g.name }))} optionFilterProp="label" showSearch />
          </Form.Item>
          <Form.Item name="overwrite" label="Перезаписывать существующие" valuePropName="checked">
            <Select options={[{ value: false, label: 'Нет' }, { value: true, label: 'Да' }]} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer title="История изменений" open={historyOpen} onClose={() => setHistoryOpen(false)} width={480}>
        {history.length === 0 ? <Text type="secondary">Нет записей</Text> : (
          <Timeline items={history.map(h => ({
            children: (
              <Space direction="vertical" size={0}>
                <Text strong>{h.action} • балл {h.score} • v{h.version}</Text>
                <Text style={{ fontSize: 12 }}>{h.text}</Text>
                <Text type="secondary" style={{ fontSize: 11 }}>{dayjs(h.changed_at).format('DD.MM.YYYY HH:mm')}</Text>
              </Space>
            ),
          }))} />
        )}
      </Drawer>
    </>
  );
}
