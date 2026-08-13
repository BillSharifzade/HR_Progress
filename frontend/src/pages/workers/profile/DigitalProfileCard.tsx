import { useState } from 'react';
import { Button, Card, Form, Input, Select, Space, Tag, Typography, theme as antdTheme } from 'antd';
import { CheckOutlined, CloseOutlined, EditOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { getWorkerProfile, saveWorkerProfile, type ProfilePayload } from '../../../api/workers';
import { Field } from '../../../components/Field';
import {
  CAREER_GOALS, DEVELOPMENT_DIRECTIONS, EDUCATION_LEVELS, EXPERIENCE_BANDS,
  INTERNAL_PROJECT_OPTIONS, LEARNING_FORMATS, LEARNING_HOURS_BANDS, MOBILITY_OPTIONS,
  PROFESSIONAL_INTERESTS, RELOCATION_OPTIONS, TEACHING_OPTIONS, withCurrent,
} from './options';

/** Renders a string[] as tags, or nothing when empty so Field shows its dash. */
function TagList({ items, color }: { items?: string[] | null; color?: string }) {
  if (!items || items.length === 0) return null;
  return (
    <Space size={[4, 4]} wrap>
      {items.map((v) => <Tag key={v} color={color}>{v}</Tag>)}
    </Space>
  );
}

/**
 * The structured half of the questionnaire. Kept as its own card with its own
 * edit toggle because it is a separate resource (PUT /workers/{id}/profile),
 * not part of the core worker record the page header edits.
 */
export function DigitalProfileCard({ workerId, canEdit }: { workerId: string; canEdit: boolean }) {
  const { token } = antdTheme.useToken();
  const qc = useQueryClient();
  const [form] = Form.useForm();
  const [editing, setEditing] = useState(false);

  const { data: profile } = useQuery({
    queryKey: ['worker-profile', workerId],
    queryFn: () => getWorkerProfile(workerId),
  });

  const save = useMutation({
    mutationFn: (payload: ProfilePayload) => saveWorkerProfile(workerId, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['worker-profile', workerId] });
      setEditing(false);
    },
  });

  const startEdit = () => {
    form.setFieldsValue({
      education_levels: profile?.education_levels ?? [],
      institution: profile?.institution ?? undefined,
      specialty: profile?.specialty ?? undefined,
      prior_experience_band: profile?.prior_experience_band ?? undefined,
      career_goal: profile?.career_goal ?? undefined,
      development_directions: profile?.development_directions ?? [],
      mobility_readiness: profile?.mobility_readiness ?? undefined,
      relocation_readiness: profile?.relocation_readiness ?? undefined,
      internal_projects_readiness: profile?.internal_projects_readiness ?? undefined,
      teaching_readiness: profile?.teaching_readiness ?? undefined,
      professional_interests: profile?.professional_interests ?? [],
      learning_formats: profile?.learning_formats ?? [],
      learning_hours_band: profile?.learning_hours_band ?? undefined,
    });
    setEditing(true);
  };

  const handleSave = async () => {
    const v = await form.validateFields();
    save.mutate({
      education_levels: v.education_levels ?? [],
      institution: v.institution || null,
      specialty: v.specialty || null,
      prior_experience_band: v.prior_experience_band || null,
      career_goal: v.career_goal || null,
      development_directions: v.development_directions ?? [],
      mobility_readiness: v.mobility_readiness || null,
      relocation_readiness: v.relocation_readiness || null,
      internal_projects_readiness: v.internal_projects_readiness || null,
      teaching_readiness: v.teaching_readiness || null,
      professional_interests: v.professional_interests ?? [],
      learning_formats: v.learning_formats ?? [],
      learning_hours_band: v.learning_hours_band || null,
    });
  };

  const extra = !canEdit ? null : editing ? (
    <Space>
      <Button size="small" icon={<CloseOutlined />} onClick={() => { form.resetFields(); setEditing(false); }}>
        Отмена
      </Button>
      <Button size="small" type="primary" icon={<CheckOutlined />} loading={save.isPending} onClick={handleSave}>
        Сохранить
      </Button>
    </Space>
  ) : (
    <Button size="small" icon={<EditOutlined />} onClick={startEdit}>Редактировать</Button>
  );

  const single = (name: string, list: string[], current?: string | null) => (
    <Form.Item name={name} noStyle>
      <Select allowClear showSearch size="small" style={{ width: 320 }}
        options={withCurrent(list, current)} placeholder="Не указано" />
    </Form.Item>
  );
  const multi = (name: string, list: string[]) => (
    <Form.Item name={name} noStyle>
      {/* `tags` mode: the form allowed free-text additions alongside its options */}
      <Select mode="tags" allowClear size="small" style={{ width: '100%', maxWidth: 520 }}
        options={list.map((v) => ({ value: v, label: v }))} placeholder="Не указано" />
    </Form.Item>
  );

  return (
    <Form form={form} component={false}>
      <Card
        title="Цифровой профиль"
        size="small"
        extra={extra}
        styles={{ body: { paddingTop: 4 } }}
      >
        {profile?.submitted_at && !editing && (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            Из анкеты от {new Date(profile.submitted_at).toLocaleDateString('ru-RU')}
          </Typography.Text>
        )}

        <div style={{ marginTop: 8, fontSize: 12, color: token.colorTextTertiary, textTransform: 'uppercase', letterSpacing: 0.4 }}>
          Образование
        </div>
        <Field label="Уровень образования" editing={editing}
          value={<TagList items={profile?.education_levels} color="geekblue" />}
          control={multi('education_levels', EDUCATION_LEVELS)} />
        <Field label="Учебное заведение" editing={editing} value={profile?.institution}
          control={<Form.Item name="institution" noStyle><Input size="small" style={{ maxWidth: 420 }} /></Form.Item>} />
        <Field label="Специальность" editing={editing} value={profile?.specialty}
          control={<Form.Item name="specialty" noStyle><Input size="small" style={{ maxWidth: 420 }} /></Form.Item>} />

        <div style={{ marginTop: 16, fontSize: 12, color: token.colorTextTertiary, textTransform: 'uppercase', letterSpacing: 0.4 }}>
          Опыт и карьера
        </div>
        <Field label="Стаж до компании" editing={editing}
          value={profile?.prior_experience_band && <Tag>{profile.prior_experience_band}</Tag>}
          control={single('prior_experience_band', EXPERIENCE_BANDS, profile?.prior_experience_band)} />
        <Field label="Карьерная цель" editing={editing} value={profile?.career_goal}
          control={single('career_goal', CAREER_GOALS, profile?.career_goal)} />
        <Field label="Направления развития" editing={editing}
          value={<TagList items={profile?.development_directions} color="blue" />}
          control={multi('development_directions', DEVELOPMENT_DIRECTIONS)} />

        <div style={{ marginTop: 16, fontSize: 12, color: token.colorTextTertiary, textTransform: 'uppercase', letterSpacing: 0.4 }}>
          Мобильность
        </div>
        <Field label="Переход в другой дпт." editing={editing} value={profile?.mobility_readiness}
          control={single('mobility_readiness', MOBILITY_OPTIONS, profile?.mobility_readiness)} />
        <Field label="Готовность к релокации" editing={editing} value={profile?.relocation_readiness}
          control={single('relocation_readiness', RELOCATION_OPTIONS, profile?.relocation_readiness)} />
        <Field label="Внутренние проекты" editing={editing} value={profile?.internal_projects_readiness}
          control={single('internal_projects_readiness', INTERNAL_PROJECT_OPTIONS, profile?.internal_projects_readiness)} />
        <Field label="Готов обучать коллег" editing={editing}
          value={profile?.teaching_readiness && (
            <Tag color={profile.teaching_readiness === 'да' ? 'success'
              : profile.teaching_readiness === 'нет' ? 'default' : 'warning'}>
              {profile.teaching_readiness}
            </Tag>
          )}
          control={single('teaching_readiness', TEACHING_OPTIONS, profile?.teaching_readiness)} />

        <div style={{ marginTop: 16, fontSize: 12, color: token.colorTextTertiary, textTransform: 'uppercase', letterSpacing: 0.4 }}>
          Обучение и развитие
        </div>
        <Field label="Проф. интересы" editing={editing}
          value={<TagList items={profile?.professional_interests} color="purple" />}
          control={multi('professional_interests', PROFESSIONAL_INTERESTS)} />
        <Field label="Формат обучения" editing={editing}
          value={<TagList items={profile?.learning_formats} />}
          control={multi('learning_formats', LEARNING_FORMATS)} />
        <Field label="Часов в месяц" editing={editing}
          value={profile?.learning_hours_band && <Tag color="cyan">{profile.learning_hours_band}</Tag>}
          control={single('learning_hours_band', LEARNING_HOURS_BANDS, profile?.learning_hours_band)} />
      </Card>
    </Form>
  );
}
