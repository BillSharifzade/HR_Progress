import { useEffect, useState, useMemo, useRef } from 'react';
import {
  Modal,
  Select,
  Table,
  InputNumber,
  Typography,
  Space,
  Button,
  Spin,
  Alert,
  Tag,
  Tooltip,
  theme as antdTheme,
  message,
} from 'antd';
import { StarFilled, LeftOutlined, RightOutlined, CommentOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';

import { listEmployees, getPeriodWithScores, upsertScoresBulk } from '../../api/competency';
import type { AssessmentPeriod, AssessmentScore, Employee, Requirement, ParticipantRole } from '../../types';
import { AssessorRoleLabel } from '../../types';
import { CommentModal } from './CommentModal';

const { Text } = Typography;

// Critical (key) competencies are shown in purple; red is reserved for the
// >4 divergence flag (two role marks disagreeing by more than 4 points).
const CRITICAL_COLOR = '#722ed1';
const CRITICAL_BG = '#f9f0ff';

const ASSESSOR_ROLES = ['HEAD', 'DEPT_HEAD', 'HRA', 'DCR_HEAD'] as const;
type AssessorRole = typeof ASSESSOR_ROLES[number];

interface DeptComp {
  key: string;
  competency_id: string;
  competency_name: string;
}

interface Props {
  period: AssessmentPeriod | null;
  deptId: string | null;
  requirements: Requirement[];
  onClose: () => void;
}

export function PeriodScoringModal({ period, deptId, requirements, onClose }: Props) {
  const [employees, setEmployees] = useState<Employee[]>([]);
  const [selectedEmployeeId, setSelectedEmployeeId] = useState<string | null>(null);
  const [allScores, setAllScores] = useState<AssessmentScore[]>([]);
  const [scores, setScores] = useState<Record<string, number | null>>({});
  const [comments, setComments] = useState<Record<string, string>>({});
  const [commentCell, setCommentCell] = useState<{ competencyId: string; role: AssessorRole } | null>(null);
  const autoOpened = useRef<Record<string, number>>({});
  const [loadingData, setLoadingData] = useState(false);
  const [saving, setSaving] = useState(false);
  const [messageApi, contextHolder] = message.useMessage();
  const { token } = antdTheme.useToken();

  const deptComps = useMemo<DeptComp[]>(() => {
    const seen = new Set<string>();
    return requirements
      .filter(r => { if (seen.has(r.competency_id)) return false; seen.add(r.competency_id); return true; })
      .map(r => ({ key: r.competency_id, competency_id: r.competency_id, competency_name: r.competency_name }));
  }, [requirements]);

  // Build requirement lookup: competency_id → grade_level → { required_min, is_key }
  const reqLookup = useMemo(() => {
    const m: Record<string, Record<number, { required_min: number | null; is_key: boolean }>> = {};
    for (const r of requirements) {
      if (!m[r.competency_id]) m[r.competency_id] = {};
      m[r.competency_id][r.grade_level] = { required_min: r.required_min, is_key: r.is_key };
    }
    return m;
  }, [requirements]);

  // Load employees + all period scores when modal opens
  useEffect(() => {
    if (!period || !deptId) return;
    setSelectedEmployeeId(null);
    setScores({});
    setAllScores([]);
    setLoadingData(true);
    Promise.all([listEmployees(deptId), getPeriodWithScores(period.id)])
      .then(([emps, data]) => {
        setEmployees(emps ?? []);
        setAllScores(data.scores ?? []);
      })
      .catch(() => messageApi.error('Не удалось загрузить данные периода'))
      .finally(() => setLoadingData(false));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [period?.id, deptId]);

  // Populate score grid when employee is selected
  useEffect(() => {
    if (!selectedEmployeeId) { setScores({}); setComments({}); return; }
    const state: Record<string, number | null> = {};
    const cstate: Record<string, string> = {};
    for (const s of allScores.filter(s => s.employee_id === selectedEmployeeId)) {
      const key = `${s.competency_id}:${s.assessor_role}`;
      state[key] = s.score;
      cstate[key] = s.feedback ?? '';
    }
    setScores(state);
    setComments(cstate);
    autoOpened.current = {};
  }, [selectedEmployeeId, allScores]);

  const handleSave = async () => {
    if (!selectedEmployeeId || !period) return;
    setSaving(true);
    try {
      const payload = Object.entries(scores)
        .filter(([, v]) => v !== null)
        .map(([key, score]) => {
          const [competency_id, assessor_role] = key.split(':');
          return {
            employee_id: selectedEmployeeId, competency_id, assessor_role,
            score: score as number, feedback: comments[key] ?? null,
          };
        });
      await upsertScoresBulk(period.id, payload);
      // Refresh stored scores so re-selecting the employee shows saved values
      const updated = await getPeriodWithScores(period.id);
      setAllScores(updated.scores ?? []);
      messageApi.success('Оценки сохранены');
    } catch {
      messageApi.error('Не удалось сохранить оценки');
    } finally {
      setSaving(false);
    }
  };

  const selectedEmployee = employees.find(e => e.id === selectedEmployeeId) ?? null;
  const employeeGradeLevel = selectedEmployee?.grade_level ?? null;

  const empIdx = selectedEmployeeId ? employees.findIndex(e => e.id === selectedEmployeeId) : -1;
  const hasPrevEmp = empIdx > 0;
  const hasNextEmp = empIdx >= 0 && empIdx < employees.length - 1;
  const goPrevEmp = () => { if (hasPrevEmp) setSelectedEmployeeId(employees[empIdx - 1].id); };
  const goNextEmp = () => { if (hasNextEmp) setSelectedEmployeeId(employees[empIdx + 1].id); };

  const isCriticalRow = (compId: string) =>
    employeeGradeLevel !== null && !!reqLookup[compId]?.[employeeGradeLevel]?.is_key;

  // Divergence: any two entered role marks for this competency differ by >4.
  const isDivergentRow = (compId: string) => {
    const vals = ASSESSOR_ROLES
      .map(role => scores[`${compId}:${role}`])
      .filter((v): v is number => v != null);
    if (vals.length < 2) return false;
    return Math.max(...vals) - Math.min(...vals) > 4;
  };

  const columns: ColumnsType<DeptComp> = [
    {
      title: 'Компетенция',
      dataIndex: 'competency_name',
      render: (name: string, row: DeptComp) => {
        const req = employeeGradeLevel !== null
          ? reqLookup[row.competency_id]?.[employeeGradeLevel]
          : undefined;
        const reqMin = req?.required_min;
        const isKey = !!req?.is_key;
        const divergent = isDivergentRow(row.competency_id);
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            {isKey && (
              <Tooltip title="Ключевая компетенция">
                <StarFilled style={{ color: CRITICAL_COLOR, fontSize: 14, flexShrink: 0 }} />
              </Tooltip>
            )}
            <Text strong style={{ fontSize: 13, color: isKey ? CRITICAL_COLOR : undefined }}>
              {name}
            </Text>
            {reqMin !== undefined && reqMin !== null && (
              <Text type="secondary" style={{ fontSize: 11 }}>min {reqMin}</Text>
            )}
            {divergent && (
              <Tooltip title="Расхождение оценок между ролями более 4 баллов">
                <Tag color="red" style={{ fontSize: 10, marginInlineStart: 'auto' }}>Расхождение</Tag>
              </Tooltip>
            )}
          </div>
        );
      },
    },
    ...ASSESSOR_ROLES.map((role: AssessorRole) => ({
      title: <div style={{ fontSize: 11, textAlign: 'center', lineHeight: 1.3 }}>{AssessorRoleLabel[role]}</div>,
      key: role,
      width: 112,
      align: 'center' as const,
      render: (_: unknown, row: DeptComp) => {
        const key = `${row.competency_id}:${role}`;
        const hasComment = !!comments[key];
        const openComment = () => setCommentCell({ competencyId: row.competency_id, role });
        return (
          <div style={{ display: 'inline-flex', alignItems: 'center', gap: 2 }}>
            <InputNumber
              value={scores[key] ?? null}
              onChange={val => setScores(prev => ({ ...prev, [key]: val as number | null }))}
              onBlur={() => {
                const v = scores[key];
                if (v != null && autoOpened.current[key] !== v) {
                  autoOpened.current[key] = v;
                  openComment();
                }
              }}
              controls={false}
              min={0} max={10} step={0.1} size="small" style={{ width: 52 }}
              placeholder="—"
              disabled={!selectedEmployeeId}
            />
            <Tooltip title={hasComment ? 'Комментарий задан' : 'Добавить комментарий'}>
              <Button
                type="text" size="small"
                icon={<CommentOutlined style={{ color: hasComment ? token.colorPrimary : token.colorTextQuaternary }} />}
                onClick={openComment}
                disabled={!selectedEmployeeId}
              />
            </Tooltip>
          </div>
        );
      },
    })),
  ];

  return (
    <>
      {contextHolder}
      <Modal
        title={period?.title}
        open={!!period}
        onCancel={onClose}
        width={860}
        centered
        destroyOnClose
        footer={
          <Space>
            <Button onClick={onClose}>Закрыть</Button>
            <Button
              type="primary"
              loading={saving}
              disabled={!selectedEmployeeId}
              onClick={handleSave}
            >
              Сохранить
            </Button>
          </Space>
        }
      >
        {period && (
          <div style={{ marginBottom: 16 }}>
            <Space>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {dayjs(period.period_start).format('DD.MM.YYYY')} — {dayjs(period.period_end).format('DD.MM.YYYY')}
              </Text>
              <Tag color={period.is_active ? 'green' : 'default'} style={{ fontSize: 11 }}>
                {period.is_active ? 'Активен' : 'Завершён'}
              </Tag>
            </Space>
          </div>
        )}

        {loadingData ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : (
          <>
            <div style={{ marginBottom: 16 }}>
              <Space align="center">
                <Text>Сотрудник:</Text>
                <Select
                  placeholder="Выберите сотрудника"
                  style={{ width: 320 }}
                  value={selectedEmployeeId}
                  onChange={id => setSelectedEmployeeId(id)}
                  options={employees.map(e => ({
                    value: e.id,
                    label: e.grade_name ? `${e.full_name} — ${e.grade_name}` : e.full_name,
                  }))}
                  notFoundContent="Нет сотрудников в департаменте"
                />
                <Tooltip title="Предыдущий сотрудник">
                  <Button icon={<LeftOutlined />} onClick={goPrevEmp} disabled={!hasPrevEmp} />
                </Tooltip>
                <Tooltip title="Следующий сотрудник">
                  <Button icon={<RightOutlined />} onClick={goNextEmp} disabled={!hasNextEmp} />
                </Tooltip>
              </Space>
            </div>

            {deptComps.length === 0 ? (
              <Alert type="info" message="Для этого департамента не настроена матрица компетенций" />
            ) : (
              <>
                <style>{`
                  .scoring-row-critical > td {
                    background: ${CRITICAL_BG} !important;
                  }
                  .scoring-row-critical > td:first-child {
                    box-shadow: inset 3px 0 0 ${CRITICAL_COLOR};
                  }
                  .scoring-row-divergent > td {
                    background: ${token.colorErrorBg} !important;
                  }
                  .scoring-row-divergent > td:first-child {
                    box-shadow: inset 3px 0 0 ${token.colorError};
                  }
                `}</style>
                <Table
                  dataSource={deptComps}
                  columns={columns}
                  pagination={false}
                  size="small"
                  scroll={{ x: 700 }}
                  rowClassName={row =>
                    isDivergentRow(row.competency_id)
                      ? 'scoring-row-divergent'
                      : isCriticalRow(row.competency_id)
                        ? 'scoring-row-critical'
                        : ''
                  }
                />
              </>
            )}
          </>
        )}
      </Modal>

      {commentCell && (
        <CommentModal
          open={!!commentCell}
          onClose={() => setCommentCell(null)}
          isAdmin
          workerId={selectedEmployeeId!}
          competencyId={commentCell.competencyId}
          competencyName={deptComps.find(d => d.competency_id === commentCell.competencyId)?.competency_name ?? ''}
          deptId={deptId}
          gradeId={selectedEmployee?.grade_id ?? null}
          initialRole={commentCell.role as ParticipantRole}
          entries={ASSESSOR_ROLES.map(role => ({
            role: role as ParticipantRole,
            score: scores[`${commentCell.competencyId}:${role}`] ?? null,
            feedback: comments[`${commentCell.competencyId}:${role}`] ?? '',
            editable: true,
          }))}
          onSave={(edits) => {
            setComments(prev => {
              const next = { ...prev };
              for (const e of edits) next[`${commentCell.competencyId}:${e.role}`] = e.feedback;
              return next;
            });
          }}
        />
      )}
    </>
  );
}
