import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert, Button, Card, Empty, Input, InputNumber, List, Space, Spin, Tabs, Tag, Tooltip,
  Typography, message,
} from 'antd';
import { ArrowLeftOutlined, SaveOutlined, StarFilled, CommentOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import dayjs from 'dayjs';

import { PageHeader } from '../../components/PageHeader';
import { PageSkeleton } from '../../components/PageSkeleton';
import {
  listMyAssessmentPeriods, listEmployees, listRequirements, listMyScoresIn, upsertScore,
} from '../../api/competency';
import { CommentModal } from '../competency/CommentModal';
import type { Employee, ParticipantRole } from '../../types';
import { ParticipantRoleLabel } from '../../types';
import { useAuth } from '../../auth/useAuth';

const { Text } = Typography;

const ROLE_COLOR: Record<ParticipantRole, string> = {
  HEAD:      'geekblue',
  DEPT_HEAD: 'purple',
  HRA:       'cyan',
  DCR_HEAD:  'orange',
  ASSESSOR:  'green',
};

export function MyPeriodScoringPage() {
  const { periodId } = useParams<{ periodId: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const [msg, ctx] = message.useMessage();

  // Period meta (from /me/assessment-periods/)
  const { data: myPeriods = [], isLoading: loadingPeriods } = useQuery({
    queryKey: ['my-assessment-periods'],
    queryFn: listMyAssessmentPeriods,
  });
  const period = useMemo(
    () => myPeriods.find(p => p.period_id === periodId) ?? null,
    [myPeriods, periodId],
  );

  const deptId = period?.department_id ?? null;

  const { data: employees = [], isLoading: loadingEmps } = useQuery({
    queryKey: ['period-employees', deptId],
    queryFn: () => listEmployees(deptId!),
    enabled: !!deptId,
  });

  const { data: requirements = [], isLoading: loadingReqs } = useQuery({
    queryKey: ['period-requirements', deptId],
    queryFn: () => listRequirements(deptId!),
    enabled: !!deptId,
  });

  const { data: myScores = [], isLoading: loadingScores, refetch: refetchScores } = useQuery({
    queryKey: ['my-scores', periodId],
    queryFn: () => listMyScoresIn(periodId!),
    enabled: !!periodId,
  });

  const myRoles = period?.roles ?? [];
  const [activeRole, setActiveRole] = useState<ParticipantRole | null>(null);
  useEffect(() => {
    if (!activeRole && myRoles.length > 0) setActiveRole(myRoles[0]);
  }, [myRoles, activeRole]);

  // Worker filtering by role. A user never scores themselves.
  const visibleWorkers = useMemo<Employee[]>(() => {
    if (!activeRole) return [];
    const notSelf = (e: Employee) => e.id !== user?.id;
    if (activeRole === 'HEAD') {
      const sec = user?.scope_section_ids?.[0] ?? user?.section_id;
      if (!sec) return [];
      return employees.filter(e => e.section_id === sec && notSelf(e));
    }
    return employees.filter(notSelf);
  }, [activeRole, employees, user]);

  const [search, setSearch] = useState('');
  const filteredWorkers = useMemo(
    () => visibleWorkers.filter(e => e.full_name.toLowerCase().includes(search.toLowerCase())),
    [visibleWorkers, search],
  );

  const [selectedWorkerId, setSelectedWorkerId] = useState<string | null>(null);
  useEffect(() => {
    if (filteredWorkers.length > 0 && !filteredWorkers.find(w => w.id === selectedWorkerId)) {
      setSelectedWorkerId(filteredWorkers[0].id);
    }
    if (filteredWorkers.length === 0) setSelectedWorkerId(null);
  }, [filteredWorkers, selectedWorkerId]);

  const selectedWorker = filteredWorkers.find(w => w.id === selectedWorkerId) ?? null;

  // Competencies (deduped from requirements) + requirement lookup
  const competencyRows = useMemo(() => {
    const seen = new Set<string>();
    const out: { competency_id: string; competency_name: string; kind: string }[] = [];
    for (const r of requirements) {
      if (seen.has(r.competency_id)) continue;
      seen.add(r.competency_id);
      out.push({
        competency_id:   r.competency_id,
        competency_name: r.competency_name,
        kind:            r.competency_kind,
      });
    }
    out.sort((a, b) => {
      if (a.kind !== b.kind) return a.kind.localeCompare(b.kind);
      return a.competency_name.localeCompare(b.competency_name, 'ru');
    });
    return out;
  }, [requirements]);

  const reqLookup = useMemo(() => {
    const m: Record<string, Record<number, { required_min: number | null; is_key: boolean }>> = {};
    for (const r of requirements) {
      (m[r.competency_id] ??= {})[r.grade_level] = { required_min: r.required_min, is_key: r.is_key };
    }
    return m;
  }, [requirements]);

  // Pending edits: { `${worker_id}:${competency_id}:${role}`: value }
  const [draft, setDraft] = useState<Record<string, number | null>>({});
  // Final interpretation text edits (FR-AS7.2.2), keyed like scores.
  const [draftText, setDraftText] = useState<Record<string, string>>({});

  const scoreKey = (wid: string, cid: string, role: ParticipantRole) => `${wid}:${cid}:${role}`;

  // Competency whose comment modal is open (role = activeRole), and auto-open
  // tracking so the modal pops once per committed mark value.
  const [commentComp, setCommentComp] = useState<string | null>(null);
  const autoOpened = useRef<Record<string, number>>({});

  const currentText = (cid: string): string => {
    if (!selectedWorker || !activeRole) return '';
    const k = scoreKey(selectedWorker.id, cid, activeRole);
    if (k in draftText) return draftText[k];
    const existing = myScores.find(s =>
      s.employee_id === selectedWorker.id && s.competency_id === cid && s.assessor_role === activeRole,
    );
    return existing?.feedback ?? '';
  };

  const currentScore = (cid: string): number | null => {
    if (!selectedWorker || !activeRole) return null;
    const k = scoreKey(selectedWorker.id, cid, activeRole);
    if (k in draft) return draft[k];
    const existing = myScores.find(s =>
      s.employee_id === selectedWorker.id &&
      s.competency_id === cid &&
      s.assessor_role === activeRole,
    );
    return existing?.score ?? null;
  };

  const dirtyKeys = Object.keys(draft);
  const pendingCount = new Set([...dirtyKeys, ...Object.keys(draftText)]).size;
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    if (!periodId) return;
    // Union of keys that changed score or comment text.
    const keys = new Set<string>([...dirtyKeys, ...Object.keys(draftText)]);
    if (keys.size === 0) return;
    setSaving(true);
    try {
      for (const k of keys) {
        const [wid, cid, role] = k.split(':');
        const scoreVal = k in draft
          ? draft[k]
          : myScores.find(s => s.employee_id === wid && s.competency_id === cid && s.assessor_role === role)?.score ?? null;
        await upsertScore(periodId, {
          employee_id: wid, competency_id: cid, assessor_role: role,
          score: scoreVal, feedback: draftText[k] ?? null,
        });
      }
      setDraft({});
      setDraftText({});
      await refetchScores();
      msg.success('Сохранено');
    } catch {
      msg.error('Не удалось сохранить');
    } finally {
      setSaving(false);
    }
  };

  const workerIdx = selectedWorker
    ? filteredWorkers.findIndex(w => w.id === selectedWorker.id)
    : -1;
  const hasPrevWorker = workerIdx > 0;
  const hasNextWorker = workerIdx >= 0 && workerIdx < filteredWorkers.length - 1;

  const goPrevWorker = () => {
    if (hasPrevWorker) setSelectedWorkerId(filteredWorkers[workerIdx - 1].id);
  };
  const goNextWorker = () => {
    if (hasNextWorker) setSelectedWorkerId(filteredWorkers[workerIdx + 1].id);
  };

  if (loadingPeriods) return <PageSkeleton type="profile" />;
  if (!period) {
    return (
      <Card>
        <Empty description="Период не найден или вы не назначены оценщиком" />
        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <Button onClick={() => navigate('/assessments')}>Назад к списку</Button>
        </div>
      </Card>
    );
  }

  const dataReady = !loadingEmps && !loadingReqs && !loadingScores;

  return (
    <>
      {ctx}
      <PageHeader
        title={period.title}
        subtitle={
          <Space size={6} wrap>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {dayjs(period.period_start).format('DD.MM.YYYY')} — {dayjs(period.period_end).format('DD.MM.YYYY')}
            </Text>
            {period.department && (
              <Tag color="default" style={{ marginInlineStart: 8 }}>{period.department}</Tag>
            )}
            <Tag color={period.is_active ? 'green' : 'default'}>
              {period.is_active ? 'Активен' : 'Завершён'}
            </Tag>
          </Space>
        }
        extra={
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/assessments')}>
            К списку периодов
          </Button>
        }
      />

      {myRoles.length === 0 && (
        <Alert type="warning" message="У вас нет ролей в этом периоде" />
      )}

      {myRoles.length > 0 && (
        <Tabs
          activeKey={activeRole ?? undefined}
          onChange={(k) => { setActiveRole(k as ParticipantRole); setDraft({}); }}
          items={myRoles.map(role => ({
            key: role,
            label: (
              <Space size={4}>
                <Tag color={ROLE_COLOR[role]}>{ParticipantRoleLabel[role]}</Tag>
              </Space>
            ),
            children: dataReady ? (
              <div style={{ display: 'flex', gap: 16, alignItems: 'flex-start' }}>
                {/* Worker list */}
                <Card
                  size="small"
                  style={{ width: 320, flexShrink: 0 }}
                  styles={{ body: { padding: 0 } }}
                  title={
                    <Input.Search
                      placeholder="Поиск сотрудника"
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                      allowClear
                      size="small"
                    />
                  }
                >
                  {filteredWorkers.length === 0 ? (
                    <Empty
                      style={{ margin: 24 }}
                      description={
                        role === 'HEAD'
                          ? 'В вашем отделе нет активных сотрудников'
                          : 'Список пуст'
                      }
                    />
                  ) : (
                    <List
                      size="small"
                      dataSource={filteredWorkers}
                      style={{ maxHeight: '60vh', overflowY: 'auto' }}
                      renderItem={(w) => {
                        const sel = w.id === selectedWorkerId;
                        return (
                          <List.Item
                            onClick={() => setSelectedWorkerId(w.id)}
                            style={{
                              cursor: 'pointer',
                              padding: '8px 12px',
                              background: sel ? 'rgba(31,94,255,0.08)' : 'transparent',
                              borderInlineStart: sel ? '3px solid #1F5EFF' : '3px solid transparent',
                            }}
                          >
                            <div style={{ width: '100%' }}>
                              <div style={{ fontWeight: sel ? 600 : 400, fontSize: 13 }}>{w.full_name}</div>
                              {w.grade_name && (
                                <Text type="secondary" style={{ fontSize: 11 }}>{w.grade_name}</Text>
                              )}
                            </div>
                          </List.Item>
                        );
                      }}
                    />
                  )}
                </Card>

                {/* Score grid for selected worker */}
                <Card
                  size="small"
                  style={{ flex: 1 }}
                  title={selectedWorker ? (
                    <Space>
                      <Text strong>{selectedWorker.full_name}</Text>
                      {selectedWorker.grade_name && (
                        <Tag>{selectedWorker.grade_name}</Tag>
                      )}
                    </Space>
                  ) : 'Выберите сотрудника'}
                  extra={selectedWorker && (
                    <Space>
                      <Button
                        type="primary"
                        size="small"
                        icon={<SaveOutlined />}
                        loading={saving}
                        disabled={pendingCount === 0}
                        onClick={handleSave}
                      >
                        Сохранить ({pendingCount})
                      </Button>
                      <Button size="small" onClick={goPrevWorker} disabled={!hasPrevWorker}>
                        ← Предыдущий
                      </Button>
                      <Button size="small" onClick={goNextWorker} disabled={!hasNextWorker}>
                        Следующий →
                      </Button>
                    </Space>
                  )}
                >
                  {!selectedWorker ? (
                    <Empty description="Нет выбранного сотрудника" />
                  ) : competencyRows.length === 0 ? (
                    <Empty description="Для этого департамента нет требований" />
                  ) : (
                    <div style={{ maxHeight: '60vh', overflowY: 'auto' }}>
                      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                        <thead>
                          <tr style={{ borderBottom: '1px solid #f0f0f0', fontSize: 12 }}>
                            <th style={{ textAlign: 'left', padding: '8px 4px' }}>Компетенция</th>
                            <th style={{ width: 100, textAlign: 'center', padding: '8px 4px' }}>Мин.</th>
                            <th style={{ width: 140, textAlign: 'center', padding: '8px 4px' }}>Моя оценка</th>
                          </tr>
                        </thead>
                        <tbody>
                          {competencyRows.map((c) => {
                            const grade = selectedWorker.grade_level ?? 0;
                            const req = reqLookup[c.competency_id]?.[grade];
                            const sc = currentScore(c.competency_id);
                            const wid = selectedWorker.id;
                            const hasComment = !!currentText(c.competency_id);
                            const key = activeRole ? scoreKey(wid, c.competency_id, activeRole) : '';
                            const openComment = () => setCommentComp(c.competency_id);
                            return (
                              <tr key={c.competency_id} style={{ borderBottom: '1px solid #f5f5f5' }}>
                                <td style={{ padding: '8px 4px' }}>
                                  <Space size={4}>
                                    <Tag color={c.kind === 'LK' ? 'blue' : c.kind === 'UK' ? 'purple' : 'gold'} style={{ fontSize: 10 }}>
                                      {c.kind}
                                    </Tag>
                                    <Text>{c.competency_name}</Text>
                                    {req?.is_key && (
                                      <Tooltip title="Ключевая компетенция">
                                        <StarFilled style={{ color: '#722ed1', fontSize: 12 }} />
                                      </Tooltip>
                                    )}
                                  </Space>
                                </td>
                                <td style={{ textAlign: 'center', padding: '8px 4px' }}>
                                  {req?.required_min != null ? (
                                    <Text>{req.required_min}</Text>
                                  ) : (
                                    <Text type="secondary">—</Text>
                                  )}
                                </td>
                                <td style={{ textAlign: 'center', padding: '8px 4px' }}>
                                  <Space size={2}>
                                    <InputNumber
                                      min={1}
                                      max={10}
                                      step={0.1}
                                      precision={1}
                                      controls={false}
                                      value={sc ?? undefined}
                                      placeholder="1–10"
                                      style={{ width: 64 }}
                                      onChange={(v) => {
                                        if (!selectedWorker || !activeRole) return;
                                        setDraft(d => ({
                                          ...d,
                                          [scoreKey(selectedWorker.id, c.competency_id, activeRole)]: v ?? null,
                                        }));
                                      }}
                                      onBlur={() => {
                                        if (sc != null && key && autoOpened.current[key] !== sc) {
                                          autoOpened.current[key] = sc;
                                          openComment();
                                        }
                                      }}
                                    />
                                    <Tooltip title={hasComment ? 'Комментарий задан' : 'Добавить комментарий'}>
                                      <Button
                                        type="text" size="small"
                                        icon={<CommentOutlined style={{ color: hasComment ? '#1F5EFF' : '#bfbfbf' }} />}
                                        onClick={openComment}
                                        disabled={sc == null}
                                      />
                                    </Tooltip>
                                  </Space>
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  )}
                </Card>
              </div>
            ) : (
              <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
            ),
          }))}
        />
      )}

      {commentComp && selectedWorker && activeRole && (
        <CommentModal
          open={!!commentComp}
          onClose={() => setCommentComp(null)}
          isAdmin={user?.roles.includes('HR_ADMIN') ?? false}
          workerId={selectedWorker.id}
          competencyId={commentComp}
          competencyName={competencyRows.find(c => c.competency_id === commentComp)?.competency_name ?? ''}
          deptId={deptId}
          gradeId={selectedWorker.grade_id ?? null}
          initialRole={activeRole}
          entries={[{
            role: activeRole,
            score: currentScore(commentComp),
            feedback: currentText(commentComp),
            editable: true,
          }]}
          onSave={(edits) => {
            setDraftText(d => {
              const next = { ...d };
              for (const e of edits) next[scoreKey(selectedWorker.id, commentComp, e.role)] = e.feedback;
              return next;
            });
          }}
        />
      )}
    </>
  );
}
