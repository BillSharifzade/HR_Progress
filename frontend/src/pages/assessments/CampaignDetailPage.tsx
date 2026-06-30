import { useMemo, useState } from 'react';
import {
  Alert, Button, Card, Col, Empty, InputNumber, Popconfirm, Row, Select, Space, Statistic,
  Table, Tabs, Tag, Timeline, Typography, message,
} from 'antd';
import { ArrowLeftOutlined, ReloadOutlined, TeamOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import dayjs from 'dayjs';

import { PageHeader } from '../../components/PageHeader';
import { PageSkeleton } from '../../components/PageSkeleton';
import {
  getPeriodWithScores, setCriteria, listCompetencies, listAssessees, addAssessees, removeAssessee,
  listAssesseeAssessors, setAssesseeAssessors, listUsersWithRole, transitionPeriod,
  listGroups, regenerateGroups, moveGroupMember, confirmGroups, listGroupJournal, listAllDepartments,
} from '../../api/competency';
import { listWorkers, listSections } from '../../api/workers';
import type { Assessee, CampaignStatus, Criterion, LearningGroup } from '../../types';
import { CampaignStatusColor, CampaignStatusLabel } from '../../types';

const { Text } = Typography;

const TRANSITIONS: Record<CampaignStatus, { to: CampaignStatus; label: string; danger?: boolean }[]> = {
  draft:        [{ to: 'assigned', label: 'Назначить участников' }],
  assigned:     [{ to: 'in_progress', label: 'Запустить оценку' }, { to: 'draft', label: 'Вернуть в черновик' }],
  in_progress:  [{ to: 'admin_review', label: 'Отправить на проверку' }],
  admin_review: [{ to: 'confirmed', label: 'Подтвердить результаты' }, { to: 'in_progress', label: 'Вернуть на пересмотр', danger: true }],
  confirmed:    [{ to: 'published', label: 'Опубликовать' }, { to: 'admin_review', label: 'Вернуть на проверку', danger: true }],
  published:    [],
};

export function CampaignDetailPage() {
  const { periodId } = useParams<{ periodId: string }>();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [msg, ctx] = message.useMessage();

  const pid = periodId!;
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ['period', pid] });
    qc.invalidateQueries({ queryKey: ['admin-periods'] });
  };

  const { data: pw, isLoading } = useQuery({ queryKey: ['period', pid], queryFn: () => getPeriodWithScores(pid) });
  const { data: competencies = [] } = useQuery({ queryKey: ['competencies'], queryFn: listCompetencies });

  const transition = useMutation({
    mutationFn: (to: CampaignStatus) => transitionPeriod(pid, to),
    onSuccess: () => { msg.success('Статус обновлён'); invalidate(); },
    onError: (e: any) => msg.error(e?.response?.data?.error?.message ?? 'Ошибка перехода'),
  });

  if (isLoading || !pw) return <PageSkeleton type="profile" />;
  const period = pw.period;
  const status = period.status;

  return (
    <>
      {ctx}
      <PageHeader
        title={period.title}
        subtitle={
          <Space wrap>
            <Tag color={CampaignStatusColor[status]}>{CampaignStatusLabel[status]}</Tag>
            <Text type="secondary">
              {dayjs(period.period_start).format('DD.MM.YYYY')} — {dayjs(period.period_end).format('DD.MM.YYYY')}
            </Text>
            <Text type="secondary">Размер группы: {period.group_size}</Text>
          </Space>
        }
        extra={
          <Space>
            {TRANSITIONS[status].map(t => (
              <Popconfirm
                key={t.to}
                title={`${t.label}?`}
                onConfirm={() => transition.mutate(t.to)}
                okText="Да"
                cancelText="Отмена"
              >
                <Button type={t.danger ? 'default' : 'primary'} danger={t.danger} loading={transition.isPending}>
                  {t.label}
                </Button>
              </Popconfirm>
            ))}
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/admin/assessments')}>К списку</Button>
          </Space>
        }
      />

      <Tabs
        items={[
          { key: 'criteria', label: 'Критерии', children: <CriteriaTab pid={pid} criteria={pw.criteria ?? []} competencies={competencies} onChange={invalidate} /> },
          { key: 'assessees', label: 'Участники', children: <AssesseesTab pid={pid} /> },
          { key: 'assessors', label: 'Асессоры', children: <AssessorsTab pid={pid} /> },
          { key: 'groups', label: 'Группы обучения', children: <GroupsTab pid={pid} groupSize={period.group_size} /> },
          { key: 'journal', label: 'Журнал групп', children: <JournalTab pid={pid} /> },
        ]}
      />
    </>
  );
}

// ── Criteria tab (FR-AS3) ────────────────────────────────────────────────────
function CriteriaTab({ pid, criteria, competencies, onChange }: {
  pid: string; criteria: Criterion[]; competencies: { id: string; name: string; kind: string }[]; onChange: () => void;
}) {
  const [msg, ctx] = message.useMessage();
  const [rows, setRows] = useState(() => criteria.map(c => ({ competency_id: c.competency_id, min_score: c.min_score ?? null })));
  const [selected, setSelected] = useState<string[]>(criteria.map(c => c.competency_id));

  const save = useMutation({
    mutationFn: () => setCriteria(pid, selected.map(cid => ({
      competency_id: cid,
      min_score: rows.find(r => r.competency_id === cid)?.min_score ?? null,
    }))),
    onSuccess: () => { msg.success('Критерии сохранены'); onChange(); },
    onError: () => msg.error('Не удалось сохранить'),
  });

  const compName = (id: string) => competencies.find(c => c.id === id)?.name ?? id;

  return (
    <Card>
      {ctx}
      <Space direction="vertical" style={{ width: '100%' }}>
        <Select
          mode="multiple" allowClear style={{ width: '100%' }} placeholder="Выберите компетенции-критерии"
          value={selected}
          onChange={(v) => {
            setSelected(v);
            setRows(prev => v.map(cid => prev.find(r => r.competency_id === cid) ?? { competency_id: cid, min_score: null }));
          }}
          options={competencies.map(c => ({ value: c.id, label: `${c.name} (${c.kind})` }))}
          optionFilterProp="label"
        />
        <Table
          rowKey="competency_id" size="small" pagination={false}
          dataSource={selected.map(cid => ({ competency_id: cid }))}
          columns={[
            { title: 'Критерий', key: 'name', render: (_: unknown, r: { competency_id: string }) => compName(r.competency_id) },
            {
              title: 'Проходной балл (1–10)', key: 'min', width: 200,
              render: (_: unknown, r: { competency_id: string }) => (
                <InputNumber
                  min={1} max={10}
                  value={rows.find(x => x.competency_id === r.competency_id)?.min_score ?? undefined}
                  onChange={(v) => setRows(prev => prev.map(x => x.competency_id === r.competency_id ? { ...x, min_score: v ?? null } : x))}
                />
              ),
            },
          ]}
        />
        <Button type="primary" onClick={() => save.mutate()} loading={save.isPending} disabled={selected.length === 0}>
          Сохранить критерии
        </Button>
      </Space>
    </Card>
  );
}

// ── Assessees tab (FR-AS2) ───────────────────────────────────────────────────
function AssesseesTab({ pid }: { pid: string }) {
  const qc = useQueryClient();
  const [msg, ctx] = message.useMessage();
  const { data: assessees = [] } = useQuery({ queryKey: ['assessees', pid], queryFn: () => listAssessees(pid) });
  const { data: departments = [] } = useQuery({ queryKey: ['all-departments'], queryFn: listAllDepartments });
  const { data: sections = [] } = useQuery({ queryKey: ['sections'], queryFn: () => listSections() });
  const { data: workers = [] } = useQuery({ queryKey: ['workers'], queryFn: () => listWorkers() });

  const [deptIds, setDeptIds] = useState<string[]>([]);
  const [sectionIds, setSectionIds] = useState<string[]>([]);
  const [userIds, setUserIds] = useState<string[]>([]);

  const add = useMutation({
    mutationFn: () => addAssessees(pid, { department_ids: deptIds, section_ids: sectionIds, user_ids: userIds }),
    onSuccess: (r) => {
      msg.success(`Добавлено участников: ${r.added}`);
      setDeptIds([]); setSectionIds([]); setUserIds([]);
      qc.invalidateQueries({ queryKey: ['assessees', pid] });
    },
    onError: () => msg.error('Не удалось добавить'),
  });
  const remove = useMutation({
    mutationFn: (userId: string) => removeAssessee(pid, userId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['assessees', pid] }),
  });

  return (
    <Row gutter={16}>
      {ctx}
      <Col span={10}>
        <Card title="Назначить участников" size="small">
          <Space direction="vertical" style={{ width: '100%' }}>
            <Select mode="multiple" allowClear placeholder="По департаментам" style={{ width: '100%' }}
              value={deptIds} onChange={setDeptIds}
              options={departments.map(d => ({ value: d.id, label: d.name }))} optionFilterProp="label" />
            <Select mode="multiple" allowClear placeholder="По отделам" style={{ width: '100%' }}
              value={sectionIds} onChange={setSectionIds}
              options={sections.map(s => ({ value: s.id, label: s.name }))} optionFilterProp="label" />
            <Select mode="multiple" allowClear placeholder="Индивидуально" style={{ width: '100%' }}
              value={userIds} onChange={setUserIds}
              options={workers.map(w => ({ value: w.id, label: w.full_name }))} optionFilterProp="label" />
            <Button type="primary" onClick={() => add.mutate()} loading={add.isPending}
              disabled={!deptIds.length && !sectionIds.length && !userIds.length}>
              Добавить
            </Button>
          </Space>
        </Card>
      </Col>
      <Col span={14}>
        <Card title={`Участники (${assessees.length})`} size="small">
          <Table rowKey="id" size="small" dataSource={assessees} pagination={{ pageSize: 10 }}
            columns={[
              { title: 'ФИО', dataIndex: 'full_name', key: 'name' },
              { title: 'Грейд', dataIndex: 'grade_name', key: 'grade', render: (g?: string) => g ?? '—' },
              { title: 'Статус', dataIndex: 'status', key: 'status', render: () => <Tag color="blue">Участник</Tag> },
              {
                title: '', key: 'x', width: 60,
                render: (_: unknown, r: Assessee) => (
                  <Popconfirm title="Убрать?" onConfirm={() => remove.mutate(r.user_id)} okText="Да" cancelText="Нет">
                    <Button type="link" danger size="small">Убрать</Button>
                  </Popconfirm>
                ),
              },
            ]}
          />
        </Card>
      </Col>
    </Row>
  );
}

// ── Assessors tab (FR-AS4) ───────────────────────────────────────────────────
function AssessorsTab({ pid }: { pid: string }) {
  const qc = useQueryClient();
  const [msg, ctx] = message.useMessage();
  const { data: assessees = [] } = useQuery({ queryKey: ['assessees', pid], queryFn: () => listAssessees(pid) });
  const { data: mappings = [] } = useQuery({ queryKey: ['assessee-assessors', pid], queryFn: () => listAssesseeAssessors(pid) });
  const { data: assessors = [] } = useQuery({ queryKey: ['assessor-users'], queryFn: () => listUsersWithRole('ASSESSOR') });

  const byAssessee = useMemo(() => {
    const m: Record<string, string[]> = {};
    for (const x of mappings) (m[x.assessee_user_id] ??= []).push(x.assessor_user_id);
    return m;
  }, [mappings]);

  const save = useMutation({
    mutationFn: (p: { assesseeId: string; ids: string[] }) => setAssesseeAssessors(pid, p.assesseeId, p.ids),
    onSuccess: () => { msg.success('Сохранено'); qc.invalidateQueries({ queryKey: ['assessee-assessors', pid] }); },
    onError: () => msg.error('Не удалось сохранить'),
  });

  return (
    <Card>
      {ctx}
      {assessors.length === 0 && (
        <Alert type="warning" showIcon style={{ marginBottom: 12 }}
          message="Нет пользователей с ролью «Ассессор». Назначьте роль в разделе сотрудников." />
      )}
      <Table rowKey="id" size="small" dataSource={assessees} pagination={{ pageSize: 12 }}
        columns={[
          { title: 'Участник', dataIndex: 'full_name', key: 'name', width: 260 },
          {
            title: 'Асессоры', key: 'assessors',
            render: (_: unknown, r: Assessee) => (
              <Select
                mode="multiple" allowClear style={{ width: '100%' }} placeholder="Назначить асессоров"
                value={byAssessee[r.user_id] ?? []}
                onChange={(ids) => save.mutate({ assesseeId: r.user_id, ids })}
                options={assessors.map(a => ({ value: a.id, label: a.full_name }))}
                optionFilterProp="label"
              />
            ),
          },
        ]}
      />
    </Card>
  );
}

// ── Groups tab (FR-AS13) ─────────────────────────────────────────────────────
function GroupsTab({ pid, groupSize }: { pid: string; groupSize: number }) {
  const qc = useQueryClient();
  const [msg, ctx] = message.useMessage();
  const [size, setSize] = useState(groupSize);
  const { data: groups = [] } = useQuery({ queryKey: ['groups', pid], queryFn: () => listGroups(pid) });

  const refresh = () => qc.invalidateQueries({ queryKey: ['groups', pid] });

  const regen = useMutation({
    mutationFn: () => regenerateGroups(pid, size),
    onSuccess: () => { msg.success('Группы переформированы'); refresh(); },
    onError: (e: any) => msg.error(e?.response?.data?.error?.message ?? 'Ошибка'),
  });
  const move = useMutation({
    mutationFn: (p: { userId: string; toGroupId: string }) => moveGroupMember(pid, p.userId, p.toGroupId),
    onSuccess: () => { msg.success('Перемещено'); refresh(); },
    onError: () => msg.error('Не удалось переместить'),
  });
  const confirm = useMutation({
    mutationFn: () => confirmGroups(pid),
    onSuccess: () => { msg.success('Группы подтверждены'); refresh(); },
  });

  const groupOptions = groups.map(g => ({ value: g.id, label: `Группа ${g.group_no}` }));

  return (
    <>
      {ctx}
      <Space style={{ marginBottom: 16 }}>
        <Text>Размер группы:</Text>
        <InputNumber min={1} max={100} value={size} onChange={(v) => setSize(v ?? 12)} />
        <Button icon={<ReloadOutlined />} onClick={() => regen.mutate()} loading={regen.isPending}>
          Сформировать заново
        </Button>
        <Popconfirm title="Подтвердить распределение групп?" onConfirm={() => confirm.mutate()} okText="Да" cancelText="Нет">
          <Button type="primary">Подтвердить группы</Button>
        </Popconfirm>
      </Space>

      {groups.length === 0 ? (
        <Empty description="Группы ещё не сформированы. Подтвердите кампанию или нажмите «Сформировать заново»." />
      ) : (
        <Row gutter={[16, 16]}>
          {groups.map((g: LearningGroup) => (
            <Col span={12} key={g.id}>
              <Card
                size="small"
                title={<Space><TeamOutlined />{`Группа ${g.group_no}`}{g.confirmed && <Tag color="green">подтв.</Tag>}</Space>}
                extra={<Text type="secondary">{g.members.length} чел.</Text>}
              >
                <Row gutter={8} style={{ marginBottom: 8 }}>
                  <Col span={12}>
                    <Statistic title="Диапазон баллов"
                      value={`${(g.score_min ?? 0).toFixed(2)} – ${(g.score_max ?? 0).toFixed(2)}`}
                      valueStyle={{ fontSize: 16 }} />
                  </Col>
                  <Col span={12}>
                    <Statistic title="Сильная сторона"
                      value={g.strength_name ? `${g.strength_name} (${(g.strength_score ?? 0).toFixed(1)})` : '—'}
                      valueStyle={{ fontSize: 14, color: '#52c41a' }} />
                  </Col>
                </Row>
                <div style={{ marginBottom: 8 }}>
                  <Text type="secondary" style={{ fontSize: 12 }}>Приоритетные зоны развития:</Text>
                  <div>
                    {g.dev_zones.length === 0 ? <Text type="secondary">—</Text> :
                      g.dev_zones.map(z => (
                        <Tag color="orange" key={z.competency_id}>{z.competency_name} — {z.avg_score.toFixed(1)}</Tag>
                      ))}
                  </div>
                </div>
                <Table
                  rowKey="id" size="small" pagination={false} showHeader={false}
                  dataSource={g.members}
                  columns={[
                    { title: '#', dataIndex: 'position', key: 'pos', width: 36, render: (p: number) => <Text type="secondary">{p}</Text> },
                    { title: 'ФИО', dataIndex: 'full_name', key: 'name' },
                    { title: 'Балл', dataIndex: 'avg_score', key: 'avg', width: 60, render: (a: number) => a.toFixed(2) },
                    {
                      title: '', key: 'move', width: 130,
                      render: (_: unknown, m) => (
                        <Select
                          size="small" style={{ width: 120 }} value={g.id}
                          options={groupOptions}
                          onChange={(toGroupId) => { if (toGroupId !== g.id) move.mutate({ userId: m.user_id, toGroupId }); }}
                        />
                      ),
                    },
                  ]}
                />
              </Card>
            </Col>
          ))}
        </Row>
      )}
    </>
  );
}

// ── Journal tab (FR-AS13.11) ─────────────────────────────────────────────────
function JournalTab({ pid }: { pid: string }) {
  const { data: entries = [] } = useQuery({ queryKey: ['group-journal', pid], queryFn: () => listGroupJournal(pid) });
  if (entries.length === 0) return <Empty description="Журнал пуст" />;
  return (
    <Card>
      <Timeline
        items={entries.map(e => ({
          children: (
            <Space direction="vertical" size={0}>
              <Text strong>{e.action}</Text>
              {e.detail && <Text type="secondary" style={{ fontSize: 12 }}>{e.detail}</Text>}
              <Text type="secondary" style={{ fontSize: 11 }}>{dayjs(e.at).format('DD.MM.YYYY HH:mm')}</Text>
            </Space>
          ),
        }))}
      />
    </Card>
  );
}
