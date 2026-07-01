import { useEffect, useState, useCallback, useMemo, type CSSProperties } from 'react';
import {
  Typography,
  Select,
  Spin,
  Alert,
  Table,
  Tag,
  Tabs,
  Card,
  Modal,
  Form,
  DatePicker,
  Input,
  InputNumber,
  Button,
  Space,
  Tooltip,
  Descriptions,
  Popconfirm,
  message,
  theme,
} from 'antd';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, StarFilled, StarOutlined,
  DownOutlined, RightOutlined, HolderOutlined, RollbackOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import {
  DndContext, closestCenter, KeyboardSensor, PointerSensor,
  useSensor, useSensors, type DragEndEvent,
} from '@dnd-kit/core';
import {
  arrayMove, SortableContext, sortableKeyboardCoordinates,
  useSortable, verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

import {
  listCompetencies,
  listAllDepartments,
  listGrades,
  listRequirements,
  listPeriods,
  createPeriod,
  createCompetency,
  updateCompetency,
  deleteCompetency,
  reorderCompetencies,
  createDepartment,
  updateDepartment,
  deleteDepartment,
  upsertRequirements,
  addPeriodParticipants,
  listUsersWithRole,
} from '../../api/competency';
import { listSections, createSection, updateSection, deleteSection } from '../../api/workers';
import type { Section } from '../../types';
import { useAuth } from '../../auth/useAuth';
import { PageHeader } from '../../components/PageHeader';
import { BRAND_PRIMARY } from '../../theme';
import { PeriodScoringModal } from './PeriodScoringModal';
import type {
  Competency,
  Department,
  Grade,
  Requirement,
  AssessmentPeriod,
  CompetencyKind,
} from '../../types';
import { CompetencyKindLabel } from '../../types';

const { Text } = Typography;

const KIND_COLOR: Record<CompetencyKind, string> = {
  LK: 'geekblue',
  UK: 'purple',
  PK: 'green',
};

const KIND_HEX: Record<CompetencyKind, string> = {
  LK: BRAND_PRIMARY,
  UK: '#722ed1',
  PK: '#52c41a',
};

const GRADES = [
  { level: 1, name: 'Стажёр' },
  { level: 2, name: 'Специалист' },
  { level: 3, name: 'Ведущий Специалист' },
  { level: 4, name: 'Главный Специалист' },
];

function buildReqMap(reqs: Requirement[]): Record<string, Record<number, Requirement>> {
  const map: Record<string, Record<number, Requirement>> = {};
  for (const r of reqs) {
    if (!map[r.competency_id]) map[r.competency_id] = {};
    map[r.competency_id][r.grade_level] = r;
  }
  return map;
}

function scoreColor(score: number | null): string {
  if (score === null) return '#d9d9d9';
  if (score <= 3) return '#52c41a';
  if (score <= 5) return BRAND_PRIMARY;
  if (score <= 7) return '#fa8c16';
  return '#f5222d';
}

function ScoreBadge({ score, isKey }: { score: number | null; isKey: boolean }) {
  if (score === null) return <Text type="secondary" style={{ fontSize: 13 }}>—</Text>;

  if (isKey) {
    return (
      <Tooltip title="Ключевая компетенция">
        <span style={{
          position: 'relative',
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: 38,
          height: 38,
          flexShrink: 0,
        }}>
          <StarFilled style={{ fontSize: 38, color: scoreColor(score) }} />
          <span style={{
            position: 'absolute',
            fontSize: 13,
            fontWeight: 800,
            color: '#fff',
            lineHeight: 1,
            userSelect: 'none',
            transform: 'translateY(1px)',
          }}>
            {score}
          </span>
        </span>
      </Tooltip>
    );
  }

  return (
    <span style={{
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      width: 30,
      height: 30,
      borderRadius: '50%',
      backgroundColor: scoreColor(score),
      color: '#fff',
      fontSize: 14,
      fontWeight: 700,
      lineHeight: 1,
      userSelect: 'none',
      flexShrink: 0,
    }}>
      {score}
    </span>
  );
}

function SortableCompetencyItem({
  comp, isAdmin, bgContainer, borderColor, onDetail, onEdit, onDelete,
}: {
  comp: Competency;
  isAdmin: boolean;
  bgContainer: string;
  borderColor: string;
  onDetail: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: comp.id });
  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.55 : 1,
    boxShadow: isDragging ? '0 6px 20px rgba(0,0,0,0.18)' : undefined,
    display: 'flex',
    alignItems: 'center',
    gap: 16,
    padding: '12px 16px',
    background: bgContainer,
    borderRadius: 8,
    border: `1px solid ${borderColor}`,
    borderLeft: `3px solid ${KIND_HEX[comp.kind]}`,
    cursor: 'pointer',
    position: 'relative',
    zIndex: isDragging ? 2 : 0,
  };
  return (
    <div ref={setNodeRef} style={style} onClick={onDetail}>
      {isAdmin && (
        <span
          {...attributes}
          {...listeners}
          onClick={e => e.stopPropagation()}
          style={{
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            width: 24, height: 28, color: 'rgba(0,0,0,0.35)',
            cursor: 'grab', touchAction: 'none', flexShrink: 0,
          }}
          title="Перетащите для изменения порядка"
        >
          <HolderOutlined />
        </span>
      )}
      <div style={{ width: 130, flexShrink: 0 }}>
        <Tag color={KIND_COLOR[comp.kind]} style={{ fontSize: 11, marginBottom: 4, display: 'block', width: 'fit-content' }}>
          {CompetencyKindLabel[comp.kind]}
        </Tag>
        <Text code style={{ fontSize: 11 }}>{comp.code}</Text>
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <Text strong style={{ display: 'block', marginBottom: 2 }}>{comp.name}</Text>
        {comp.description && (
          <Text type="secondary" ellipsis style={{ fontSize: 12, display: 'block' }}>
            {comp.description}
          </Text>
        )}
      </div>
      <Space align="center">
        <Tag color={comp.is_active ? 'green' : 'default'} style={{ fontSize: 11 }}>
          {comp.is_active ? 'Активна' : 'Неактивна'}
        </Tag>
        {isAdmin && (
          <Space size={4} onClick={e => e.stopPropagation()}>
            <Tooltip title="Редактировать">
              <Button type="text" size="small" icon={<EditOutlined />} onClick={onEdit} />
            </Tooltip>
            <Popconfirm
              title="Удалить компетенцию?"
              description="Это действие необратимо."
              okText="Удалить"
              okButtonProps={{ danger: true }}
              cancelText="Отмена"
              onConfirm={onDelete}
            >
              <Tooltip title="Удалить">
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Tooltip>
            </Popconfirm>
          </Space>
        )}
      </Space>
    </div>
  );
}

interface MatrixTableRow {
  key: string;
  competency: Competency;
  reqMap: Record<number, Requirement>;
}

type CompetencyFormValues = {
  kind: CompetencyKind;
  name: string;
  description?: string;
  why_important?: string;
  is_active?: boolean;
};

type DeptFormValues = {
  name: string;
  description?: string;
  is_active?: boolean;
};

type EditCell = { required_min: number | null; is_key: boolean };

export function CompetencyMatrixPage() {
  const { user } = useAuth();
  const isAdmin = user?.roles.includes('HR_ADMIN') ?? false;
  const { token } = theme.useToken();

  const [departments, setDepartments] = useState<Department[]>([]);
  const [competencies, setCompetencies] = useState<Competency[]>([]);
  const [grades, setGrades] = useState<Grade[]>([]);
  const [requirements, setRequirements] = useState<Requirement[]>([]);
  const [periods, setPeriods] = useState<AssessmentPeriod[]>([]);
  const [selectedDeptId, setSelectedDeptId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // period create modal
  const [createPeriodOpen, setCreatePeriodOpen] = useState(false);
  const [creatingPeriod, setCreatingPeriod] = useState(false);
  const [periodForm] = Form.useForm();
  const [periodAssessors, setPeriodAssessors] = useState<{ id: string; full_name: string }[]>([]);

  // competency detail modal
  const [detailComp, setDetailComp] = useState<Competency | null>(null);

  // competency create/edit modal
  const [compFormOpen, setCompFormOpen] = useState(false);
  const [editingComp, setEditingComp] = useState<Competency | null>(null);
  const [savingComp, setSavingComp] = useState(false);
  const [compForm] = Form.useForm<CompetencyFormValues>();

  // matrix inline editing
  const [matrixEditing, setMatrixEditing] = useState(false);
  const [matrixEditState, setMatrixEditState] = useState<Record<string, EditCell>>({});
  // Competencies the admin removed from THIS department's matrix during edit.
  // Excluded from the save payload; since UpsertRequirements is replace-all,
  // they drop out of the dept matrix. Values are preserved so removal is undoable.
  const [removedComps, setRemovedComps] = useState<Set<string>>(new Set());
  const [savingMatrix, setSavingMatrix] = useState(false);

  // period scoring modal
  const [scoringPeriod, setScoringPeriod] = useState<AssessmentPeriod | null>(null);

  // department create/edit modal
  const [deptFormOpen, setDeptFormOpen] = useState(false);
  const [editingDept, setEditingDept] = useState<Department | null>(null);
  const [savingDept, setSavingDept] = useState(false);
  const [deptForm] = Form.useForm<DeptFormValues>();

  // sections per department
  const [expandedDepts, setExpandedDepts] = useState<Set<string>>(new Set());
  const [closingDepts, setClosingDepts] = useState<Set<string>>(new Set());
  const [sectionsMap, setSectionsMap] = useState<Record<string, Section[]>>({});
  const [loadingSections, setLoadingSections] = useState<Set<string>>(new Set());
  const [sectionFormOpen, setSectionFormOpen] = useState(false);
  const [sectionFormDeptId, setSectionFormDeptId] = useState<string | null>(null);
  const [editingSection, setEditingSection] = useState<Section | null>(null);
  const [savingSection, setSavingSection] = useState(false);
  const [sectionForm] = Form.useForm();

  const [messageApi, contextHolder] = message.useMessage();

  useEffect(() => {
    Promise.all([listCompetencies(), listAllDepartments(), listGrades()])
      .then(([comps, depts, gs]) => {
        setCompetencies(comps);
        setDepartments(depts);
        setGrades(gs);
      })
      .catch(() => setError('Не удалось загрузить данные'));
  }, []);

  // Non-admins: auto-select their scoped department on mount and lock the selector.
  const lockedDeptId = !isAdmin
    ? (user?.scope_department_ids?.[0] ?? user?.department_id ?? null)
    : null;

  useEffect(() => {
    if (lockedDeptId && lockedDeptId !== selectedDeptId) {
      setSelectedDeptId(lockedDeptId);
      loadDeptData(lockedDeptId);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lockedDeptId]);

  const loadDeptData = useCallback(async (deptId: string) => {
    setLoading(true);
    setError(null);
    try {
      const [reqs, ps] = await Promise.all([listRequirements(deptId), listPeriods(deptId)]);
      setRequirements(reqs);
      setPeriods(ps);
    } catch {
      setError('Не удалось загрузить данные департамента');
    } finally {
      setLoading(false);
    }
  }, []);

  const handleDeptChange = (deptId: string) => {
    setSelectedDeptId(deptId);
    setMatrixEditing(false);
    loadDeptData(deptId);
  };

  const reqMap = useMemo(() => buildReqMap(requirements), [requirements]);

  const tableRows: MatrixTableRow[] = useMemo(
    () => competencies
      .filter(c => reqMap[c.id])
      .map(c => ({ key: c.id, competency: c, reqMap: reqMap[c.id] ?? {} })),
    [competencies, reqMap],
  );

  const gradeColumns: ColumnsType<MatrixTableRow> = GRADES.map(g => ({
    title: <div style={{ fontSize: 12, textAlign: 'center', fontWeight: 600 }}>{g.name}</div>,
    key: `grade_${g.level}`,
    width: 130,
    align: 'center' as const,
    render: (_: unknown, row: MatrixTableRow) => {
      const req = row.reqMap[g.level];
      if (!req) return <Text type="secondary" style={{ fontSize: 12 }}>—</Text>;
      return <ScoreBadge score={req.required_min} isKey={req.is_key} />;
    },
  }));

  const matrixColumns: ColumnsType<MatrixTableRow> = [
    {
      title: 'Тип',
      key: 'kind',
      width: 110,
      render: (_: unknown, row: MatrixTableRow) => (
        <Tag color={KIND_COLOR[row.competency.kind]} style={{ fontSize: 11 }}>
          {CompetencyKindLabel[row.competency.kind]}
        </Tag>
      ),
    },
    {
      title: 'Компетенция',
      key: 'name',
      ellipsis: true,
      render: (_: unknown, row: MatrixTableRow) => (
        <Tooltip title={row.competency.description ?? ''} placement="topLeft">
          <Text strong style={{ fontSize: 13 }}>{row.competency.name}</Text>
        </Tooltip>
      ),
    },
    ...gradeColumns,
  ];

  // --- Matrix edit mode ---

  const enterEditMode = () => {
    const state: Record<string, EditCell> = {};
    for (const comp of competencies) {
      for (const grade of grades) {
        state[`${comp.id}:${grade.id}`] = { required_min: null, is_key: false };
      }
    }
    for (const req of requirements) {
      state[`${req.competency_id}:${req.grade_id}`] = {
        required_min: req.required_min,
        is_key: req.is_key,
      };
    }
    setMatrixEditState(state);
    setRemovedComps(new Set());
    setMatrixEditing(true);
  };

  const toggleRemoveComp = (compId: string) => {
    setRemovedComps(prev => {
      const next = new Set(prev);
      if (next.has(compId)) next.delete(compId); else next.add(compId);
      return next;
    });
  };

  const saveMatrix = async () => {
    if (!selectedDeptId) return;
    setSavingMatrix(true);
    try {
      const payload = Object.entries(matrixEditState)
        .filter(([, v]) => v.required_min !== null || v.is_key)
        .map(([key, v]) => {
          const [competency_id, grade_id] = key.split(':');
          return { competency_id, grade_id, required_min: v.required_min, is_key: v.is_key };
        })
        // Competencies removed from this dept's matrix are dropped on save.
        .filter(p => !removedComps.has(p.competency_id));
      await upsertRequirements(selectedDeptId, payload);
      await loadDeptData(selectedDeptId);
      setMatrixEditing(false);
      messageApi.success('Требования сохранены');
    } catch {
      messageApi.error('Не удалось сохранить требования');
    } finally {
      setSavingMatrix(false);
    }
  };

  const editMatrixColumns: ColumnsType<Competency> = [
    {
      title: 'Тип',
      key: 'kind',
      width: 110,
      render: (_: unknown, comp: Competency) => (
        <Tag color={KIND_COLOR[comp.kind]} style={{ fontSize: 11 }}>
          {CompetencyKindLabel[comp.kind]}
        </Tag>
      ),
    },
    {
      title: 'Компетенция',
      key: 'name',
      ellipsis: true,
      render: (_: unknown, comp: Competency) => (
        <Text strong style={{ fontSize: 13 }}>{comp.name}</Text>
      ),
    },
    ...grades.map(grade => ({
      title: <div style={{ fontSize: 11, textAlign: 'center', fontWeight: 600, lineHeight: 1.3 }}>{grade.name}</div>,
      key: `grade_edit_${grade.id}`,
      width: 110,
      align: 'center' as const,
      render: (_: unknown, comp: Competency) => {
        const key = `${comp.id}:${grade.id}`;
        const cell: EditCell = matrixEditState[key] ?? { required_min: null, is_key: false };
        return (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2, alignItems: 'center' }}>
            <InputNumber
              value={cell.required_min}
              onChange={val => setMatrixEditState(prev => ({
                ...prev,
                [key]: { ...(prev[key] ?? { is_key: false }), required_min: val as number | null },
              }))}
              min={0} max={10} size="small" style={{ width: 58 }}
              placeholder="—"
            />
            <Button
              type="text" size="small"
              icon={cell.is_key
                ? <StarFilled style={{ color: '#faad14' }} />
                : <StarOutlined style={{ color: '#bfbfbf' }} />}
              onClick={() => setMatrixEditState(prev => ({
                ...prev,
                [key]: { ...(prev[key] ?? { required_min: null }), is_key: !cell.is_key },
              }))}
            />
          </div>
        );
      },
    })),
    {
      title: '',
      key: 'remove',
      width: 48,
      align: 'center' as const,
      render: (_: unknown, comp: Competency) => (
        removedComps.has(comp.id) ? (
          <Tooltip title="Вернуть в матрицу">
            <Button type="text" size="small" icon={<RollbackOutlined />} onClick={() => toggleRemoveComp(comp.id)} />
          </Tooltip>
        ) : (
          <Popconfirm
            title="Убрать из матрицы департамента?"
            description="Компетенция останется в каталоге, но будет удалена из матрицы этого департамента при сохранении."
            okText="Убрать"
            okButtonProps={{ danger: true }}
            cancelText="Отмена"
            onConfirm={() => toggleRemoveComp(comp.id)}
          >
            <Tooltip title="Убрать из матрицы">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        )
      ),
    },
  ];

  // --- Competency CRUD ---

  const openCreateComp = () => {
    setEditingComp(null);
    compForm.resetFields();
    compForm.setFieldValue('is_active', true);
    setCompFormOpen(true);
  };

  const openEditComp = (comp: Competency) => {
    setEditingComp(comp);
    compForm.setFieldsValue({
      kind: comp.kind,
      name: comp.name,
      description: comp.description ?? undefined,
      why_important: comp.why_important ?? undefined,
      is_active: comp.is_active,
    });
    setCompFormOpen(true);
  };

  const handleSaveComp = async (values: CompetencyFormValues) => {
    setSavingComp(true);
    try {
      const basePayload = {
        kind: values.kind,
        name: values.name,
        description: values.description ?? null,
        why_important: values.why_important ?? null,
      };
      if (editingComp) {
        const updated = await updateCompetency(editingComp.id, {
          ...basePayload,
          is_active: values.is_active ?? true,
        });
        setCompetencies(prev => prev.map(c => c.id === updated.id ? updated : c));
        messageApi.success('Компетенция обновлена');
      } else {
        const created = await createCompetency(basePayload);
        setCompetencies(prev => [...prev, created]);
        messageApi.success('Компетенция создана');
      }
      setCompFormOpen(false);
      compForm.resetFields();
    } catch {
      messageApi.error('Не удалось сохранить компетенцию');
    } finally {
      setSavingComp(false);
    }
  };

  const handleDeleteComp = async (id: string) => {
    try {
      await deleteCompetency(id);
      setCompetencies(prev => prev.filter(c => c.id !== id));
      messageApi.success('Компетенция удалена');
    } catch {
      messageApi.error('Не удалось удалить компетенцию');
    }
  };

  // --- Drag-and-drop reorder ---

  const dndSensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const handleCompetencyDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const oldIdx = competencies.findIndex(c => c.id === active.id);
    const newIdx = competencies.findIndex(c => c.id === over.id);
    if (oldIdx < 0 || newIdx < 0) return;
    const prev = competencies;
    const reordered = arrayMove(competencies, oldIdx, newIdx);
    setCompetencies(reordered);
    try {
      await reorderCompetencies(reordered.map(c => c.id));
    } catch {
      setCompetencies(prev);
      messageApi.error('Не удалось изменить порядок');
    }
  };

  // --- Section helpers ---

  const loadSections = async (deptId: string) => {
    setLoadingSections(prev => new Set(prev).add(deptId));
    try {
      const list = await listSections(deptId);
      setSectionsMap(prev => ({ ...prev, [deptId]: list }));
    } finally {
      setLoadingSections(prev => { const s = new Set(prev); s.delete(deptId); return s; });
    }
  };

  const toggleDept = (deptId: string) => {
    if (expandedDepts.has(deptId)) {
      setClosingDepts(prev => new Set(prev).add(deptId));
      setTimeout(() => {
        setExpandedDepts(prev => { const n = new Set(prev); n.delete(deptId); return n; });
        setClosingDepts(prev => { const n = new Set(prev); n.delete(deptId); return n; });
      }, 160);
    } else {
      setExpandedDepts(prev => new Set(prev).add(deptId));
      if (!sectionsMap[deptId]) loadSections(deptId);
    }
  };

  const openAddSection = (deptId: string) => {
    setEditingSection(null);
    setSectionFormDeptId(deptId);
    sectionForm.resetFields();
    setSectionFormOpen(true);
  };

  const openEditSection = (section: Section) => {
    setEditingSection(section);
    setSectionFormDeptId(section.department_id);
    sectionForm.setFieldsValue({ name: section.name, description: section.description, is_active: section.is_active });
    setSectionFormOpen(true);
  };

  const handleSaveSection = async () => {
    const v = await sectionForm.validateFields();
    setSavingSection(true);
    try {
      if (editingSection) {
        const updated = await updateSection(editingSection.id, {
          name: v.name,
          description: v.description || null,
          is_active: v.is_active ?? true,
        });
        setSectionsMap(prev => ({
          ...prev,
          [updated.department_id]: (prev[updated.department_id] ?? []).map(s => s.id === updated.id ? updated : s),
        }));
      } else {
        const created = await createSection({
          department_id: sectionFormDeptId!,
          name: v.name,
          description: v.description || null,
        });
        setSectionsMap(prev => ({
          ...prev,
          [created.department_id]: [...(prev[created.department_id] ?? []), created],
        }));
      }
      setSectionFormOpen(false);
      sectionForm.resetFields();
    } catch {
      messageApi.error('Не удалось сохранить отдел');
    } finally {
      setSavingSection(false);
    }
  };

  const handleDeleteSection = async (section: Section) => {
    try {
      await deleteSection(section.id);
      setSectionsMap(prev => ({
        ...prev,
        [section.department_id]: (prev[section.department_id] ?? []).filter(s => s.id !== section.id),
      }));
    } catch {
      messageApi.error('Не удалось удалить отдел');
    }
  };

  // --- Department CRUD ---

  const openCreateDept = () => {
    setEditingDept(null);
    deptForm.resetFields();
    setDeptFormOpen(true);
  };

  const openEditDept = (dept: Department) => {
    setEditingDept(dept);
    deptForm.setFieldsValue({
      name: dept.name,
      description: dept.description ?? undefined,
      is_active: dept.is_active,
    });
    setDeptFormOpen(true);
  };

  const handleSaveDept = async (values: DeptFormValues) => {
    setSavingDept(true);
    try {
      if (editingDept) {
        const updated = await updateDepartment(editingDept.id, {
          name: values.name,
          description: values.description ?? null,
          is_active: values.is_active ?? editingDept.is_active,
        });
        setDepartments(prev => prev.map(d => d.id === updated.id ? updated : d));
        messageApi.success('Департамент обновлён');
      } else {
        const created = await createDepartment({
          name: values.name,
          description: values.description ?? null,
        });
        setDepartments(prev => [...prev, created]);
        messageApi.success('Департамент создан');
      }
      setDeptFormOpen(false);
      deptForm.resetFields();
    } catch {
      messageApi.error('Не удалось сохранить департамент');
    } finally {
      setSavingDept(false);
    }
  };

  const handleDeleteDept = async (id: string) => {
    try {
      await deleteDepartment(id);
      setDepartments(prev => prev.filter(d => d.id !== id));
      messageApi.success('Департамент удалён');
    } catch {
      messageApi.error('Не удалось удалить департамент');
    }
  };

  // --- Period create ---

  const handleCreatePeriod = async (values: {
    title: string;
    dates: [dayjs.Dayjs, dayjs.Dayjs];
    assessor_user_ids: string[];
  }) => {
    if (!selectedDeptId) return;
    if (!values.assessor_user_ids || values.assessor_user_ids.length < 2) {
      messageApi.error('Нужно минимум 2 ассессора');
      return;
    }
    setCreatingPeriod(true);
    try {
      const period = await createPeriod({
        title: values.title,
        department_id: selectedDeptId,
        period_start: values.dates[0].format('YYYY-MM-DD'),
        period_end: values.dates[1].format('YYYY-MM-DD'),
      });
      const participants = values.assessor_user_ids.map(id => ({
        user_id: id,
        role: 'ASSESSOR' as const,
      }));
      await addPeriodParticipants(period.id, participants);
      setPeriods(prev => [period, ...prev]);
      setCreatePeriodOpen(false);
      periodForm.resetFields();
      messageApi.success('Период оценки создан, ассессоры назначены');
    } catch (err) {
      const e = err as { response?: { data?: { error?: { code?: string; message?: string } } } };
      const msg = e?.response?.data?.error?.message;
      messageApi.error(msg ?? 'Не удалось создать период');
    } finally {
      setCreatingPeriod(false);
    }
  };

  // Load assessor pool when create-period modal opens.
  useEffect(() => {
    if (!createPeriodOpen) return;
    listUsersWithRole('ASSESSOR')
      .then(assessors => {
        setPeriodAssessors(assessors.map(a => ({ id: a.id, full_name: a.full_name })));
      })
      .catch(() => messageApi.warning('Не удалось загрузить список ассессоров'));
  }, [createPeriodOpen, messageApi]);

  const periodColumns: ColumnsType<AssessmentPeriod> = [
    {
      title: 'Название',
      dataIndex: 'title',
      render: (t: string) => <Text strong>{t}</Text>,
    },
    {
      title: 'Период',
      key: 'period',
      render: (_: unknown, r: AssessmentPeriod) =>
        `${dayjs(r.period_start).format('DD.MM.YYYY')} — ${dayjs(r.period_end).format('DD.MM.YYYY')}`,
    },
    {
      title: 'Статус',
      dataIndex: 'is_active',
      width: 100,
      render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? 'Активен' : 'Завершён'}</Tag>,
    },
    {
      title: 'Создан',
      dataIndex: 'created_at',
      width: 160,
      render: (v: string) => dayjs(v).format('DD.MM.YYYY HH:mm'),
    },
  ];

  const activeDepts = departments.filter(d => d.is_active);

  return (
    <>
      {contextHolder}
      <PageHeader title="Матрица Компетенций" />

      {error && <Alert type="error" message={error} style={{ marginBottom: 16 }} />}

      <Tabs
        defaultActiveKey="matrix"
        items={[
          {
            key: 'matrix',
            label: 'Матрица требований',
            children: (
              <>
                <Card style={{ marginBottom: 16 }}>
                  <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
                    <Space>
                      <Text>Департамент:</Text>
                      <Select
                        placeholder="Выберите департамент"
                        style={{ width: 300 }}
                        value={selectedDeptId ?? undefined}
                        onChange={handleDeptChange}
                        disabled={!isAdmin}
                        options={activeDepts.map(d => ({ value: d.id, label: `${d.code} — ${d.name}` }))}
                      />
                    </Space>
                    {isAdmin && selectedDeptId && !matrixEditing && (
                      <Button icon={<EditOutlined />} onClick={enterEditMode} disabled={grades.length === 0}>
                        Редактировать матрицу
                      </Button>
                    )}
                    {matrixEditing && (
                      <Space>
                        <Button onClick={() => setMatrixEditing(false)}>Отмена</Button>
                        <Button type="primary" loading={savingMatrix} onClick={saveMatrix}>
                          Сохранить
                        </Button>
                      </Space>
                    )}
                  </Space>
                </Card>

                {loading ? (
                  <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" /></div>
                ) : matrixEditing ? (
                  <>
                    <style>{`.matrix-row-removed > td { opacity: 0.4; }`}</style>
                    <Table
                      dataSource={competencies}
                      rowKey="id"
                      columns={editMatrixColumns}
                      pagination={false}
                      size="small"
                      scroll={{ x: 800 }}
                      rowClassName={comp => (removedComps.has(comp.id) ? 'matrix-row-removed' : '')}
                    />
                  </>
                ) : selectedDeptId && tableRows.length === 0 ? (
                  <Alert type="info" message="Нет данных требований для выбранного департамента" />
                ) : (
                  <Table
                    dataSource={tableRows}
                    columns={matrixColumns}
                    pagination={false}
                    size="small"
                    scroll={{ x: 800 }}
                  />
                )}
              </>
            ),
          },
          ...(isAdmin ? [
          {
            key: 'periods',
            label: 'Периоды оценки',
            children: (
              <>
                <Card style={{ marginBottom: 16 }}>
                  <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
                    <Space>
                      <Text>Департамент:</Text>
                      <Select
                        placeholder="Выберите департамент"
                        style={{ width: 300 }}
                        onChange={handleDeptChange}
                        value={selectedDeptId}
                        options={activeDepts.map(d => ({ value: d.id, label: `${d.code} — ${d.name}` }))}
                      />
                    </Space>
                    {isAdmin && selectedDeptId && (
                      <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreatePeriodOpen(true)}>
                        Создать период
                      </Button>
                    )}
                  </Space>
                </Card>

                {loading ? (
                  <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" /></div>
                ) : (
                  <Table
                    dataSource={periods}
                    columns={periodColumns}
                    rowKey="id"
                    pagination={{ pageSize: 20 }}
                    locale={{ emptyText: selectedDeptId ? 'Периодов нет' : 'Выберите департамент' }}
                    onRow={period => ({
                      onClick: () => setScoringPeriod(period),
                      style: { cursor: 'pointer' },
                    })}
                  />
                )}
              </>
            ),
          },
          {
            key: 'catalog',
            label: 'Каталог компетенций',
            children: (
              <>
                {isAdmin && (
                  <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'flex-end' }}>
                    <Button type="primary" icon={<PlusOutlined />} onClick={openCreateComp}>
                      Добавить компетенцию
                    </Button>
                  </div>
                )}
                {competencies.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" /></div>
                ) : (
                  <DndContext
                    sensors={dndSensors}
                    collisionDetection={closestCenter}
                    onDragEnd={handleCompetencyDragEnd}
                  >
                    <SortableContext items={competencies.map(c => c.id)} strategy={verticalListSortingStrategy}>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                        {competencies.map(comp => (
                          <SortableCompetencyItem
                            key={comp.id}
                            comp={comp}
                            isAdmin={isAdmin}
                            bgContainer={token.colorBgContainer}
                            borderColor={token.colorBorderSecondary}
                            onDetail={() => setDetailComp(comp)}
                            onEdit={() => openEditComp(comp)}
                            onDelete={() => handleDeleteComp(comp.id)}
                          />
                        ))}
                      </div>
                    </SortableContext>
                  </DndContext>
                )}
              </>
            ),
          },
          {
            key: 'departments',
            label: 'Департаменты',
            children: (
              <>
                {isAdmin && (
                  <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'flex-end' }}>
                    <Button type="primary" icon={<PlusOutlined />} onClick={openCreateDept}>
                      Добавить департамент
                    </Button>
                  </div>
                )}
                {departments.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" /></div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                    {departments.map(dept => {
                      const expanded = expandedDepts.has(dept.id);
                      const closing = closingDepts.has(dept.id);
                      const sections = sectionsMap[dept.id] ?? [];
                      const loadingDept = loadingSections.has(dept.id);
                      return (
                        <div key={dept.id}>
                          {/* Department row */}
                          <div
                            style={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 16,
                              padding: '12px 16px',
                              background: token.colorBgContainer,
                              borderRadius: (expanded || closing) ? `${token.borderRadiusLG}px ${token.borderRadiusLG}px 0 0` : token.borderRadiusLG,
                              border: `1px solid ${token.colorBorderSecondary}`,
                              borderLeft: `3px solid ${dept.is_active ? token.colorPrimary : token.colorBorderSecondary}`,
                              borderBottom: (expanded || closing) ? 'none' : undefined,
                            }}
                          >
                            <Button
                              type="text" size="small"
                              icon={expanded ? <DownOutlined style={{ fontSize: 11 }} /> : <RightOutlined style={{ fontSize: 11 }} />}
                              onClick={() => toggleDept(dept.id)}
                              loading={loadingDept}
                              style={{ flexShrink: 0, color: token.colorTextSecondary }}
                            />
                            <Text code style={{ width: 60, flexShrink: 0, fontSize: 12 }}>{dept.code}</Text>
                            <div style={{ flex: 1, minWidth: 0 }}>
                              <Text strong style={{ display: 'block', marginBottom: 2 }}>{dept.name}</Text>
                              {dept.description && (
                                <Text type="secondary" ellipsis style={{ fontSize: 12, display: 'block' }}>
                                  {dept.description}
                                </Text>
                              )}
                            </div>
                            <Space align="center">
                              <Tag color={dept.is_active ? 'green' : 'default'} style={{ fontSize: 11 }}>
                                {dept.is_active ? 'Активен' : 'Неактивен'}
                              </Tag>
                              {isAdmin && (
                                <Space size={4}>
                                  {!expanded && (
                                    <Tooltip title="Добавить отдел">
                                      <Button type="text" size="small" icon={<PlusOutlined />} onClick={() => { openAddSection(dept.id); toggleDept(dept.id); }} />
                                    </Tooltip>
                                  )}
                                  <Tooltip title="Редактировать">
                                    <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openEditDept(dept)} />
                                  </Tooltip>
                                  <Popconfirm
                                    title="Удалить департамент?"
                                    description="Это действие необратимо."
                                    okText="Удалить"
                                    okButtonProps={{ danger: true }}
                                    cancelText="Отмена"
                                    onConfirm={() => handleDeleteDept(dept.id)}
                                  >
                                    <Tooltip title="Удалить">
                                      <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                                    </Tooltip>
                                  </Popconfirm>
                                </Space>
                              )}
                            </Space>
                          </div>

                          {/* Sections panel */}
                          {(expanded || closing) && (
                            <div className={closing ? 'dept-section-panel-close' : 'dept-section-panel'} style={{
                              border: `1px solid ${token.colorBorderSecondary}`,
                              borderTop: 'none',
                              borderRadius: `0 0 ${token.borderRadiusLG}px ${token.borderRadiusLG}px`,
                              background: token.colorFillAlter,
                              padding: '8px 12px 12px 48px',
                            }}>
                              {loadingDept ? (
                                <div style={{ padding: '12px 0', textAlign: 'center' }}><Spin size="small" /></div>
                              ) : sections.length === 0 ? (
                                <Text type="secondary" style={{ fontSize: 13, display: 'block', padding: '8px 0' }}>
                                  Отделы не добавлены
                                </Text>
                              ) : (
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 4, marginTop: 6 }}>
                                  {sections.map(sec => (
                                    <div key={sec.id} style={{
                                      display: 'flex', alignItems: 'center', gap: 12,
                                      padding: '8px 12px',
                                      background: token.colorBgContainer,
                                      borderRadius: token.borderRadius,
                                      border: `1px solid ${token.colorBorderSecondary}`,
                                    }}>
                                      {sec.code && (
                                        <Text code style={{ fontSize: 11, flexShrink: 0 }}>{sec.code}</Text>
                                      )}
                                      <div style={{ flex: 1, minWidth: 0 }}>
                                        <Text style={{ display: 'block', fontSize: 13 }}>{sec.name}</Text>
                                        {sec.description && (
                                          <Text type="secondary" style={{ fontSize: 12 }}>{sec.description}</Text>
                                        )}
                                      </div>
                                      <Tag color={sec.is_active ? 'green' : 'default'} style={{ fontSize: 11 }}>
                                        {sec.is_active ? 'Активен' : 'Неактивен'}
                                      </Tag>
                                      {isAdmin && (
                                        <Space size={4}>
                                          <Tooltip title="Редактировать">
                                            <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openEditSection(sec)} />
                                          </Tooltip>
                                          <Popconfirm
                                            title="Удалить отдел?"
                                            description="Это действие необратимо."
                                            okText="Удалить"
                                            okButtonProps={{ danger: true }}
                                            cancelText="Отмена"
                                            onConfirm={() => handleDeleteSection(sec)}
                                          >
                                            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                                          </Popconfirm>
                                        </Space>
                                      )}
                                    </div>
                                  ))}
                                </div>
                              )}
                              {isAdmin && (
                                <Button
                                  type="dashed" size="small" icon={<PlusOutlined />}
                                  style={{ marginTop: 8 }}
                                  onClick={() => openAddSection(dept.id)}
                                >
                                  Добавить отдел
                                </Button>
                              )}
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}
              </>
            ),
          },
          ] : []),
        ]}
      />

      <PeriodScoringModal
        period={scoringPeriod}
        deptId={selectedDeptId}
        requirements={requirements}
        onClose={() => setScoringPeriod(null)}
      />

      {/* Period create modal */}
      <Modal
        title="Новый период оценки"
        open={createPeriodOpen}
        onCancel={() => { setCreatePeriodOpen(false); periodForm.resetFields(); }}
        footer={null}
        destroyOnClose
        centered
        width={560}
      >
        <Form form={periodForm} layout="vertical" onFinish={handleCreatePeriod} style={{ marginTop: 16 }}>
          <Form.Item name="title" label="Название" rules={[{ required: true, message: 'Введите название' }]}>
            <Input placeholder="Оценка Q1 2026" />
          </Form.Item>
          <Form.Item name="dates" label="Период" rules={[{ required: true, message: 'Выберите даты' }]}>
            <DatePicker.RangePicker style={{ width: '100%' }} format="DD.MM.YYYY" />
          </Form.Item>

          <Form.Item
            name="assessor_user_ids"
            label="Ассессоры (минимум 2)"
            rules={[
              { required: true, message: 'Выберите ассессоров' },
              { validator: (_, v) => (v?.length >= 2 ? Promise.resolve() : Promise.reject(new Error('Минимум 2 ассессора'))) },
            ]}
            extra={
              periodAssessors.length < 2
                ? <Text type="warning" style={{ fontSize: 12 }}>В системе меньше двух пользователей с ролью «Ассессор» — выдайте роль в разделе «Администрирование».</Text>
                : null
            }
          >
            <Select
              mode="multiple"
              showSearch
              placeholder="Выберите ассессоров"
              optionFilterProp="label"
              options={periodAssessors.map(a => ({ value: a.id, label: a.full_name }))}
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => { setCreatePeriodOpen(false); periodForm.resetFields(); }}>Отмена</Button>
              <Button type="primary" htmlType="submit" loading={creatingPeriod}>Создать</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Section create/edit modal */}
      <Modal
        title={editingSection ? 'Редактировать отдел' : 'Новый отдел'}
        open={sectionFormOpen}
        onCancel={() => { setSectionFormOpen(false); sectionForm.resetFields(); }}
        onOk={handleSaveSection}
        okText={editingSection ? 'Сохранить' : 'Создать'}
        cancelText="Отмена"
        confirmLoading={savingSection}
        destroyOnClose
        centered
      >
        <Form form={sectionForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="name" label="Название" rules={[{ required: true, message: 'Введите название' }]}>
            <Input placeholder="Название отдела" />
          </Form.Item>
          <Form.Item name="description" label="Описание">
            <Input.TextArea rows={2} placeholder="Необязательно" />
          </Form.Item>
          {editingSection && (
            <Form.Item name="is_active" label="Статус">
              <Select options={[{ value: true, label: 'Активен' }, { value: false, label: 'Неактивен' }]} />
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* Competency detail modal */}
      <Modal
        title={detailComp?.name}
        open={!!detailComp}
        onCancel={() => setDetailComp(null)}
        footer={
          isAdmin ? (
            <Space>
              <Button onClick={() => setDetailComp(null)}>Закрыть</Button>
              <Button
                type="primary"
                icon={<EditOutlined />}
                onClick={() => { openEditComp(detailComp!); setDetailComp(null); }}
              >
                Редактировать
              </Button>
            </Space>
          ) : null
        }
        width={600}
        centered
      >
        {detailComp && (
          <Descriptions column={1} bordered size="small" style={{ marginTop: 8 }}>
            <Descriptions.Item label="Код">
              <Text code>{detailComp.code}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="Тип">
              <Tag color={KIND_COLOR[detailComp.kind]}>{CompetencyKindLabel[detailComp.kind]}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Описание">
              {detailComp.description ?? <Text type="secondary">—</Text>}
            </Descriptions.Item>
            <Descriptions.Item label="Почему важно">
              {detailComp.why_important ?? <Text type="secondary">—</Text>}
            </Descriptions.Item>
            <Descriptions.Item label="Порядок сортировки">
              {detailComp.sort_order}
            </Descriptions.Item>
            <Descriptions.Item label="Статус">
              <Tag color={detailComp.is_active ? 'green' : 'default'}>
                {detailComp.is_active ? 'Активна' : 'Неактивна'}
              </Tag>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>

      {/* Competency create/edit modal */}
      <Modal
        title={editingComp ? 'Редактировать компетенцию' : 'Новая компетенция'}
        open={compFormOpen}
        onCancel={() => { setCompFormOpen(false); compForm.resetFields(); }}
        footer={null}
        destroyOnClose
        width={560}
        centered
      >
        <Form form={compForm} layout="vertical" onFinish={handleSaveComp} style={{ marginTop: 16 }}>
          <Form.Item name="kind" label="Тип" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'LK', label: 'Личностные' },
                { value: 'UK', label: 'Управленческие' },
                { value: 'PK', label: 'Профессиональные' },
              ]}
            />
          </Form.Item>
          <Form.Item name="name" label="Название" rules={[{ required: true, message: 'Введите название' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Описание">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="why_important" label="Почему важно">
            <Input.TextArea rows={3} />
          </Form.Item>
          {editingComp && (
            <Form.Item name="is_active" label="Статус">
              <Select
                options={[
                  { value: true, label: 'Активна' },
                  { value: false, label: 'Неактивна' },
                ]}
              />
            </Form.Item>
          )}
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => { setCompFormOpen(false); compForm.resetFields(); }}>Отмена</Button>
              <Button type="primary" htmlType="submit" loading={savingComp}>
                {editingComp ? 'Сохранить' : 'Создать'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Department create/edit modal */}
      <Modal
        title={editingDept ? 'Редактировать департамент' : 'Новый департамент'}
        open={deptFormOpen}
        onCancel={() => { setDeptFormOpen(false); deptForm.resetFields(); }}
        footer={null}
        destroyOnClose
        width={480}
        centered
      >
        <Form form={deptForm} layout="vertical" onFinish={handleSaveDept} style={{ marginTop: 16 }}>
          <Form.Item name="name" label="Название" rules={[{ required: true, message: 'Введите название' }]}>
            <Input placeholder="Департамент Информационных Технологий" />
          </Form.Item>
          <Form.Item name="description" label="Описание">
            <Input.TextArea rows={2} />
          </Form.Item>
          {editingDept && (
            <Form.Item name="is_active" label="Статус">
              <Select
                options={[
                  { value: true, label: 'Активен' },
                  { value: false, label: 'Неактивен' },
                ]}
              />
            </Form.Item>
          )}
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => { setDeptFormOpen(false); deptForm.resetFields(); }}>Отмена</Button>
              <Button type="primary" htmlType="submit" loading={savingDept}>
                {editingDept ? 'Сохранить' : 'Создать'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
