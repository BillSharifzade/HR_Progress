import { useState } from 'react';
import dayjs from 'dayjs';
import {
  Button, Card, Drawer, Empty, Form, Input, Popconfirm, Space, Tag, Typography,
} from 'antd';
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  deleteSurveyAnswer, listSurveyAnswers, updateSurveyAnswer, upsertSurveyAnswer,
  type SurveyAnswerPayload,
} from '../../../api/workers';
import type { WorkerSurveyAnswer } from '../../../types';

/** Slugifies a question into a stable code when HR adds one by hand. */
function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, '_')
    .replace(/^_+|_+$/g, '')
    .slice(0, 60) || `q_${Date.now()}`;
}

function AnswerDrawer({
  workerId, entry, onClose,
}: { workerId: string; entry: WorkerSurveyAnswer | null; onClose: () => void }) {
  const [form] = Form.useForm();
  const qc = useQueryClient();
  const isEdit = !!entry;

  const mut = useMutation({
    mutationFn: (v: SurveyAnswerPayload) =>
      isEdit ? updateSurveyAnswer(workerId, entry!.id, v) : upsertSurveyAnswer(workerId, v),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['worker-survey', workerId] });
      onClose();
    },
  });

  const submit = async () => {
    const v = await form.validateFields();
    mut.mutate({
      question_code: entry?.question_code ?? slugify(v.question_text),
      question_text: v.question_text,
      answer_text: v.answer_text,
    });
  };

  return (
    <Drawer
      open width={520} onClose={onClose}
      title={isEdit ? 'Изменить ответ' : 'Добавить ответ'}
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
        initialValues={entry ? { question_text: entry.question_text, answer_text: entry.answer_text } : undefined}
      >
        <Form.Item name="question_text" label="Вопрос" rules={[{ required: true, message: 'Укажите вопрос' }]}>
          <Input.TextArea rows={2} placeholder="Например: В каких проектах Вы участвовали?" />
        </Form.Item>
        <Form.Item name="answer_text" label="Ответ" rules={[{ required: true, message: 'Укажите ответ' }]}>
          <Input.TextArea rows={6} />
        </Form.Item>
      </Form>
    </Drawer>
  );
}

/**
 * «Результаты опросов» — the open-ended questionnaire answers that have no
 * dedicated profile field. Rendered as question/answer blocks rather than a
 * table because the answers are paragraphs, not values.
 */
export function SurveyTab({ workerId, canEdit }: { workerId: string; canEdit: boolean }) {
  const qc = useQueryClient();
  const [drawer, setDrawer] = useState<{ open: boolean; entry: WorkerSurveyAnswer | null }>({
    open: false, entry: null,
  });

  const { data: answers = [] } = useQuery({
    queryKey: ['worker-survey', workerId],
    queryFn: () => listSurveyAnswers(workerId),
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteSurveyAnswer(workerId, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['worker-survey', workerId] }),
  });

  const submitted = answers.find((a) => a.submitted_at)?.submitted_at;

  return (
    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 }}>
        {submitted ? (
          <Typography.Text type="secondary" style={{ fontSize: 13 }}>
            Анкета заполнена {dayjs(submitted).format('DD.MM.YYYY')}
          </Typography.Text>
        ) : <span />}
        {canEdit && (
          <Button icon={<PlusOutlined />} onClick={() => setDrawer({ open: true, entry: null })}>
            Добавить ответ
          </Button>
        )}
      </div>

      {answers.length === 0 ? (
        <Empty description="Ответы не заполнены" />
      ) : (
        <Space direction="vertical" size="middle" style={{ width: '100%' }}>
          {answers.map((a) => (
            <Card
              key={a.id} size="small"
              title={<span style={{ fontWeight: 500, whiteSpace: 'normal' }}>{a.question_text}</span>}
              extra={canEdit && (
                <Space size={0}>
                  <Button type="text" size="small" icon={<EditOutlined />}
                    onClick={() => setDrawer({ open: true, entry: a })} />
                  <Popconfirm title="Удалить ответ?" okText="Да" cancelText="Нет"
                    onConfirm={() => remove.mutate(a.id)}>
                    <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                  </Popconfirm>
                </Space>
              )}
            >
              <Typography.Paragraph style={{ marginBottom: 0, whiteSpace: 'pre-wrap' }}>
                {a.answer_text}
              </Typography.Paragraph>
              {a.source === 'form' && (
                <Tag style={{ marginTop: 8 }} color="default">из анкеты</Tag>
              )}
            </Card>
          ))}
        </Space>
      )}

      {drawer.open && (
        <AnswerDrawer
          workerId={workerId} entry={drawer.entry}
          onClose={() => setDrawer({ open: false, entry: null })}
        />
      )}
    </Space>
  );
}
