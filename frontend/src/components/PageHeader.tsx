import type { ReactNode } from 'react';
import { Typography } from 'antd';

export function PageHeader({
  title,
  subtitle,
  extra,
}: {
  title: ReactNode;
  subtitle?: ReactNode;
  extra?: ReactNode;
}) {
  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'flex-start',
        gap: 16,
        marginBottom: 24,
        minHeight: 40,
      }}
    >
      <div style={{ minWidth: 0, flex: 1 }}>
        <Typography.Title level={3} style={{ margin: 0, lineHeight: '32px' }}>
          {title}
        </Typography.Title>
        {subtitle && (
          <div style={{ marginTop: 2, fontSize: 13, color: 'rgba(0,0,0,0.55)' }}>
            {subtitle}
          </div>
        )}
      </div>
      {extra && <div style={{ flexShrink: 0 }}>{extra}</div>}
    </div>
  );
}
