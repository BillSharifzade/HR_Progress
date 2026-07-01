import { useEffect, useMemo, useState } from 'react';
import {
  Modal, Input, Button, Space, Typography, Segmented, Alert, Tag, Divider, Popconfirm, message,
} from 'antd';
import {
  lookupInterpretation, listInterpretations, upsertInterpretation, deleteInterpretation,
} from '../../api/competency';
import type { ParticipantRole } from '../../types';
import { ParticipantRoleLabel } from '../../types';
import { roundMark } from '../../utils/mark';

const { Text } = Typography;

export interface CommentEntry {
  role: ParticipantRole;
  score: number | null;
  feedback: string;
  editable: boolean;
}

interface Props {
  open: boolean;
  onClose: () => void;
  isAdmin: boolean;
  workerId: string;
  competencyId: string;
  competencyName: string;
  deptId: string | null;   // for prefilled-comment (template) CRUD
  gradeId: string | null;  // worker's grade, for template CRUD
  entries: CommentEntry[]; // one per role available in this context
  initialRole: ParticipantRole;
  // Persist edited comments (only roles whose text changed are passed).
  onSave: (edits: { role: ParticipantRole; feedback: string }[]) => void;
}

export function CommentModal({
  open, onClose, isAdmin, workerId, competencyId, competencyName,
  deptId, gradeId, entries, initialRole, onSave,
}: Props) {
  const [msg, ctx] = message.useMessage();
  const [activeRole, setActiveRole] = useState<ParticipantRole>(initialRole);
  const [drafts, setDrafts] = useState<Record<string, string>>({});

  // Prefilled-comment template (from the interpretations справочник) for the
  // active role's rounded score.
  const [template, setTemplate] = useState<{ text: string; found: boolean } | null>(null);
  // Admin-only editable template record (id present ⇒ exists, for delete).
  const [tplDraft, setTplDraft] = useState('');
  const [tplId, setTplId] = useState<string | null>(null);
  const [tplSaving, setTplSaving] = useState(false);
  const [showTplEditor, setShowTplEditor] = useState(false);

  // Reset local state only when the modal transitions to open — `entries` is a
  // fresh array on every parent render, so depending on it would wipe in-progress
  // edits whenever the parent re-renders while the modal is open.
  useEffect(() => {
    if (!open) return;
    setActiveRole(initialRole);
    setDrafts(Object.fromEntries(entries.map(e => [e.role, e.feedback])));
    setShowTplEditor(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const activeEntry = useMemo(
    () => entries.find(e => e.role === activeRole) ?? null,
    [entries, activeRole],
  );
  const rounded = activeEntry?.score != null ? roundMark(activeEntry.score) : null;

  // Fetch the template text + admin record for the active role's score.
  useEffect(() => {
    if (!open || rounded == null) { setTemplate(null); setTplId(null); setTplDraft(''); return; }
    let cancelled = false;
    lookupInterpretation(workerId, competencyId, rounded)
      .then(r => { if (!cancelled) setTemplate({ text: r.text ?? '', found: r.found }); })
      .catch(() => { if (!cancelled) setTemplate({ text: '', found: false }); });
    if (isAdmin && deptId && gradeId) {
      listInterpretations({ department_id: deptId, grade_id: gradeId, competency_id: competencyId })
        .then(list => {
          if (cancelled) return;
          const rec = list.find(i => i.score === rounded);
          setTplId(rec?.id ?? null);
          setTplDraft(rec?.text ?? '');
        })
        .catch(() => { if (!cancelled) { setTplId(null); setTplDraft(''); } });
    }
    return () => { cancelled = true; };
  }, [open, rounded, workerId, competencyId, isAdmin, deptId, gradeId]);

  const handleSave = () => {
    const edits = entries
      .filter(e => e.editable && (drafts[e.role] ?? '') !== e.feedback)
      .map(e => ({ role: e.role, feedback: drafts[e.role] ?? '' }));
    onSave(edits);
    onClose();
  };

  const saveTemplate = async () => {
    if (!deptId || !gradeId || rounded == null) return;
    setTplSaving(true);
    try {
      await upsertInterpretation({
        department_id: deptId, grade_id: gradeId, competency_id: competencyId,
        score: rounded, text: tplDraft,
      });
      setTemplate({ text: tplDraft, found: true });
      msg.success('Шаблон сохранён');
      setShowTplEditor(false);
    } catch {
      msg.error('Не удалось сохранить шаблон');
    } finally {
      setTplSaving(false);
    }
  };

  const removeTemplate = async () => {
    if (!tplId) return;
    setTplSaving(true);
    try {
      await deleteInterpretation(tplId);
      setTplId(null); setTplDraft(''); setTemplate({ text: '', found: false });
      msg.success('Шаблон удалён');
      setShowTplEditor(false);
    } catch {
      msg.error('Не удалось удалить шаблон');
    } finally {
      setTplSaving(false);
    }
  };

  const useTemplate = () => {
    if (!activeEntry?.editable || !template) return;
    setDrafts(d => ({ ...d, [activeRole]: template.text }));
  };

  return (
    <Modal
      open={open}
      onCancel={onClose}
      title={<Space size={6}><Text strong>Комментарий к оценке</Text><Text type="secondary" style={{ fontWeight: 400 }}>· {competencyName}</Text></Space>}
      width={560}
      centered
      destroyOnClose
      footer={
        <Space>
          <Button onClick={onClose}>Отмена</Button>
          <Button type="primary" onClick={handleSave}>Сохранить</Button>
        </Space>
      }
    >
      {ctx}
      {entries.length > 1 && (
        <Segmented
          block
          style={{ marginBottom: 12 }}
          value={activeRole}
          onChange={(v) => setActiveRole(v as ParticipantRole)}
          options={entries.map(e => ({
            label: ParticipantRoleLabel[e.role] + (e.score != null ? ` · ${e.score}` : ''),
            value: e.role,
          }))}
        />
      )}

      <div style={{ marginBottom: 8 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>Оценка: </Text>
        {activeEntry?.score != null
          ? <Tag color="blue">{activeEntry.score}{rounded != null && rounded !== activeEntry.score ? ` → ${rounded}` : ''}</Tag>
          : <Text type="secondary">—</Text>}
      </div>

      {template?.found && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 8, fontSize: 12 }}
          message="Шаблон-комментарий для этой оценки"
          description={<span style={{ fontSize: 12 }}>{template.text}</span>}
          action={activeEntry?.editable
            ? <Button size="small" type="link" onClick={useTemplate}>Использовать</Button>
            : undefined}
        />
      )}

      <Input.TextArea
        autoSize={{ minRows: 2, maxRows: 6 }}
        placeholder={activeEntry?.editable ? 'Комментарий к оценке' : 'Комментарий не задан'}
        value={drafts[activeRole] ?? ''}
        disabled={!activeEntry?.editable}
        onChange={(e) => setDrafts(d => ({ ...d, [activeRole]: e.target.value }))}
      />

      {isAdmin && deptId && gradeId && rounded != null && (
        <>
          <Divider style={{ margin: '14px 0 10px' }} />
          {!showTplEditor ? (
            <Button size="small" type="dashed" onClick={() => setShowTplEditor(true)}>
              {template?.found ? 'Изменить шаблон-комментарий' : 'Создать шаблон-комментарий'} для оценки {rounded}
            </Button>
          ) : (
            <div>
              <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 6 }}>
                Шаблон-комментарий для оценки {rounded} (справочник) — подставляется всем по умолчанию
              </Text>
              <Input.TextArea
                autoSize={{ minRows: 2, maxRows: 5 }}
                value={tplDraft}
                onChange={(e) => setTplDraft(e.target.value)}
                placeholder="Текст шаблона"
              />
              <Space style={{ marginTop: 8 }}>
                <Button size="small" type="primary" loading={tplSaving} onClick={saveTemplate}>Сохранить шаблон</Button>
                {tplId && (
                  <Popconfirm title="Удалить шаблон?" okText="Удалить" cancelText="Отмена"
                    okButtonProps={{ danger: true }} onConfirm={removeTemplate}>
                    <Button size="small" danger loading={tplSaving}>Удалить</Button>
                  </Popconfirm>
                )}
                <Button size="small" onClick={() => setShowTplEditor(false)}>Закрыть</Button>
              </Space>
            </div>
          )}
        </>
      )}
    </Modal>
  );
}
