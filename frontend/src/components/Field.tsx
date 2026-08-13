import { theme as antdTheme } from 'antd';

/**
 * One label/value row inside a profile card. In read mode it shows the value
 * (or a dimmed em dash when empty); in edit mode it swaps in the control.
 *
 * Extracted from WorkerProfile so the digital-profile cards render identically
 * to the personal-data card they sit beside.
 */
export function Field({
  label, value, editing, control, labelWidth = 170,
}: {
  label: string;
  value: React.ReactNode;
  editing: boolean;
  control: React.ReactNode;
  labelWidth?: number;
}) {
  const { token } = antdTheme.useToken();
  return (
    <div style={{
      display: 'flex',
      alignItems: editing ? 'center' : 'flex-start',
      padding: '10px 0',
      borderBottom: `1px solid ${token.colorBorderSecondary}`,
      gap: 12,
    }}>
      <span style={{
        width: labelWidth, flexShrink: 0, fontSize: 13,
        color: token.colorTextSecondary, userSelect: 'none', paddingTop: editing ? 0 : 2,
      }}>
        {label}
      </span>
      <div style={{ flex: 1, minWidth: 0 }}>
        {editing
          ? <div key="ctrl" className="field-control">{control}</div>
          : <span key="txt" className="field-text" style={{ color: value ? token.colorText : token.colorTextDisabled }}>{value ?? '—'}</span>}
      </div>
    </div>
  );
}
