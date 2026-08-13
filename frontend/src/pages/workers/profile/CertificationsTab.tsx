import { useState } from 'react';
import dayjs, { type Dayjs } from 'dayjs';
import {
  Button, DatePicker, Drawer, Empty, Form, Input, Popconfirm, Space, Table, Tag, Tooltip, Upload, message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  DeleteOutlined, DownloadOutlined, EditOutlined, LinkOutlined,
  PaperClipOutlined, PlusOutlined, UploadOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  createCertification, deleteCertification, deleteCertificationFile,
  downloadCertificationFile, listCertifications, updateCertification, uploadCertificationFile,
  type CertificationPayload,
} from '../../../api/workers';
import type { WorkerCertification } from '../../../types';

function fmt(d?: string | null) { return d ? dayjs(d).format('DD.MM.YYYY') : '—'; }

function humanSize(bytes?: number | null) {
  if (!bytes) return '';
  const units = ['Б', 'КБ', 'МБ'];
  let n = bytes; let i = 0;
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i += 1; }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function CertDrawer({
  workerId, entry, onClose,
}: { workerId: string; entry: WorkerCertification | null; onClose: () => void }) {
  const [form] = Form.useForm();
  const qc = useQueryClient();
  const isEdit = !!entry;

  const mut = useMutation({
    mutationFn: (v: CertificationPayload) =>
      isEdit ? updateCertification(workerId, entry!.id, v) : createCertification(workerId, v),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['certifications', workerId] });
      onClose();
    },
  });

  const submit = async () => {
    const v = await form.validateFields();
    mut.mutate({
      title: v.title,
      issued_by: v.issued_by || null,
      issued_at: v.issued_at ? (v.issued_at as Dayjs).format('YYYY-MM-DD') : null,
      expires_at: v.expires_at ? (v.expires_at as Dayjs).format('YYYY-MM-DD') : null,
      source_url: v.source_url || null,
    });
  };

  return (
    <Drawer
      open width={440} onClose={onClose}
      title={isEdit ? 'Изменить сертификат' : 'Добавить сертификат'}
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
          title: entry.title,
          issued_by: entry.issued_by,
          issued_at: entry.issued_at ? dayjs(entry.issued_at) : null,
          expires_at: entry.expires_at ? dayjs(entry.expires_at) : null,
          source_url: entry.source_url,
        } : undefined}
      >
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
        <Form.Item
          name="source_url" label="Ссылка на сертификат"
          tooltip="Ссылка и загруженный файл взаимоисключающи — прикрепление файла очистит ссылку."
          rules={[{ type: 'url', message: 'Введите корректную ссылку' }]}
        >
          <Input placeholder="https://drive.google.com/…" disabled={!!entry?.has_file} />
        </Form.Item>
      </Form>
    </Drawer>
  );
}

/**
 * Certificates, held either as a link or as an uploaded document. The two are
 * mutually exclusive server-side, so the UI shows whichever is present and
 * lets HR swap one for the other.
 */
export function CertificationsTab({ workerId, canEdit }: { workerId: string; canEdit: boolean }) {
  const qc = useQueryClient();
  const [drawer, setDrawer] = useState<{ open: boolean; entry: WorkerCertification | null }>({
    open: false, entry: null,
  });

  const { data: rows = [] } = useQuery({
    queryKey: ['certifications', workerId],
    queryFn: () => listCertifications(workerId),
  });
  const invalidate = () => qc.invalidateQueries({ queryKey: ['certifications', workerId] });

  const remove = useMutation({
    mutationFn: (id: string) => deleteCertification(workerId, id),
    onSuccess: invalidate,
  });
  const upload = useMutation({
    mutationFn: ({ id, file }: { id: string; file: File }) =>
      uploadCertificationFile(workerId, id, file),
    onSuccess: () => { invalidate(); message.success('Файл прикреплён'); },
    onError: (e: any) => {
      const code = e?.response?.data?.error?.code;
      message.error(
        code === 'UNSUPPORTED_TYPE' ? 'Неподдерживаемый формат файла'
          : code === 'TOO_LARGE' ? 'Файл слишком большой'
            : 'Не удалось загрузить файл',
      );
    },
  });
  const detach = useMutation({
    mutationFn: (id: string) => deleteCertificationFile(workerId, id),
    onSuccess: invalidate,
  });

  const columns: ColumnsType<WorkerCertification> = [
    { title: 'Сертификат', dataIndex: 'title', key: 'title', render: (v: string) => <span style={{ fontWeight: 500 }}>{v}</span> },
    { title: 'Организация', dataIndex: 'issued_by', key: 'issued_by', render: (v) => v ?? '—' },
    { title: 'Получен', key: 'issued', width: 110, render: (_, r) => fmt(r.issued_at) },
    { title: 'До', key: 'expires', width: 110, render: (_, r) => fmt(r.expires_at) },
    {
      title: 'Документ', key: 'doc', width: 230,
      render: (_, r) => {
        if (r.has_file) {
          return (
            <Space size={4}>
              <Tooltip title={`${r.file_name ?? 'файл'} · ${humanSize(r.file_size)}`}>
                <Button size="small" type="link" icon={<DownloadOutlined />}
                  onClick={() => downloadCertificationFile(workerId, r.id, r.file_name ?? 'certificate')}>
                  Скачать
                </Button>
              </Tooltip>
              {canEdit && (
                <Popconfirm title="Открепить файл?" okText="Да" cancelText="Нет"
                  onConfirm={() => detach.mutate(r.id)}>
                  <Button size="small" type="text" danger icon={<DeleteOutlined />} />
                </Popconfirm>
              )}
            </Space>
          );
        }
        if (r.source_url) {
          return (
            <Space size={4}>
              <LinkOutlined style={{ opacity: 0.5 }} />
              <a href={r.source_url} target="_blank" rel="noopener noreferrer">Открыть ссылку</a>
            </Space>
          );
        }
        return canEdit ? (
          <Upload
            showUploadList={false}
            beforeUpload={(file) => { upload.mutate({ id: r.id, file }); return false; }}
          >
            <Button size="small" type="text" icon={<UploadOutlined />}>Прикрепить</Button>
          </Upload>
        ) : <span style={{ opacity: 0.45 }}>—</span>;
      },
    },
    ...(canEdit ? [{
      key: 'actions', width: 88,
      render: (_: unknown, r: WorkerCertification) => (
        <Space size={0}>
          <Button type="text" size="small" icon={<EditOutlined />}
            onClick={() => setDrawer({ open: true, entry: r })} />
          <Popconfirm title="Удалить сертификат?" okText="Да" cancelText="Нет"
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
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Tag icon={<PaperClipOutlined />} color="default">
            PDF, JPG, PNG, DOC/DOCX — до 20 МБ
          </Tag>
          <Button icon={<PlusOutlined />} onClick={() => setDrawer({ open: true, entry: null })}>
            Добавить сертификат
          </Button>
        </div>
      )}
      <Table
        rowKey="id" size="small" columns={columns} dataSource={rows} pagination={false}
        locale={{ emptyText: <Empty description="Сертификаты не добавлены" /> }}
      />
      {drawer.open && (
        <CertDrawer
          workerId={workerId} entry={drawer.entry}
          onClose={() => setDrawer({ open: false, entry: null })}
        />
      )}
    </Space>
  );
}
