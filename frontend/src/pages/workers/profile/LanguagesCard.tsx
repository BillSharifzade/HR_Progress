import { useState } from 'react';
import { AutoComplete, Button, Card, Empty, Popconfirm, Select, Space, Tag, Tooltip, theme as antdTheme } from 'antd';
import { CheckOutlined, CloseOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import {
  deleteLanguage, listLanguages, upsertLanguage, updateLanguage,
} from '../../../api/workers';
import { CEFR_LEVELS, type CefrLevel } from '../../../types';
import { CEFR_COLORS, KNOWN_LANGUAGES } from './options';

const LEVEL_OPTIONS = CEFR_LEVELS.map((l) => ({ value: l, label: l }));

/**
 * Language proficiency as a row of chips: language name plus its CEFR level,
 * coloured by band so fluency reads at a glance. Adding a language that is
 * already present updates its level (the backend upserts on the name).
 */
export function LanguagesCard({ workerId, canEdit }: { workerId: string; canEdit: boolean }) {
  const { token } = antdTheme.useToken();
  const qc = useQueryClient();
  const [adding, setAdding] = useState(false);
  const [draftLang, setDraftLang] = useState('');
  const [draftLevel, setDraftLevel] = useState<CefrLevel>('B1');

  const { data: languages = [] } = useQuery({
    queryKey: ['worker-languages', workerId],
    queryFn: () => listLanguages(workerId),
  });

  const invalidate = () => qc.invalidateQueries({ queryKey: ['worker-languages', workerId] });

  const add = useMutation({
    mutationFn: () => upsertLanguage(workerId, { language: draftLang.trim(), level: draftLevel }),
    onSuccess: () => { invalidate(); setAdding(false); setDraftLang(''); setDraftLevel('B1'); },
  });
  const changeLevel = useMutation({
    mutationFn: ({ id, language, level }: { id: string; language: string; level: CefrLevel }) =>
      updateLanguage(workerId, id, { language, level }),
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: (id: string) => deleteLanguage(workerId, id),
    onSuccess: invalidate,
  });

  // Offer the six languages the form asked about, minus the ones already added.
  const suggestions = KNOWN_LANGUAGES
    .filter((l) => !languages.some((x) => x.language.toLowerCase() === l.toLowerCase()))
    .map((l) => ({ value: l }));

  return (
    <Card
      title="Владение языками"
      size="small"
      extra={canEdit && !adding && (
        <Button size="small" icon={<PlusOutlined />} onClick={() => setAdding(true)}>Добавить</Button>
      )}
    >
      {languages.length === 0 && !adding && (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Языки не указаны" style={{ margin: '8px 0' }} />
      )}

      <Space size={[8, 8]} wrap>
        {languages.map((l) => (
          <div key={l.id} style={{
            display: 'inline-flex', alignItems: 'center', gap: 8,
            border: `1px solid ${token.colorBorderSecondary}`,
            borderRadius: token.borderRadius, padding: '4px 8px',
          }}>
            <span style={{ fontSize: 13 }}>{l.language}</span>
            {canEdit ? (
              <Select
                size="small" value={l.level} options={LEVEL_OPTIONS} style={{ width: 72 }}
                onChange={(level) => changeLevel.mutate({ id: l.id, language: l.language, level })}
              />
            ) : (
              <Tag color={CEFR_COLORS[l.level]} style={{ marginInlineEnd: 0 }}>{l.level}</Tag>
            )}
            {canEdit && (
              <Popconfirm title="Удалить язык?" okText="Да" cancelText="Нет"
                onConfirm={() => remove.mutate(l.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>
            )}
          </div>
        ))}
      </Space>

      {adding && (
        <Space style={{ marginTop: languages.length ? 12 : 0 }} wrap>
          <AutoComplete
            autoFocus
            size="small"
            style={{ width: 200 }}
            placeholder="Язык"
            value={draftLang}
            options={suggestions}
            onChange={setDraftLang}
            filterOption={(input, option) =>
              (option?.value ?? '').toLowerCase().includes(input.toLowerCase())}
          />
          <Select size="small" style={{ width: 80 }} value={draftLevel}
            options={LEVEL_OPTIONS} onChange={setDraftLevel} />
          <Tooltip title={!draftLang.trim() ? 'Введите название языка' : ''}>
            <Button size="small" type="primary" icon={<CheckOutlined />}
              disabled={!draftLang.trim()} loading={add.isPending}
              onClick={() => add.mutate()} />
          </Tooltip>
          <Button size="small" icon={<CloseOutlined />}
            onClick={() => { setAdding(false); setDraftLang(''); }} />
        </Space>
      )}
    </Card>
  );
}
