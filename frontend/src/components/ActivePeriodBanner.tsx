import { useEffect } from 'react';
import { App, Button } from 'antd';
import { BellOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';

import { listMyAssessmentPeriods } from '../api/competency';
import { useAuth } from '../auth/useAuth';

const SESSION_KEY_PREFIX = 'active-period-toast-shown:';

/**
 * Fires a single 5s toast on first mount per session when the user has any
 * active assessment period(s). Replaces the previous per-period banner stack.
 * The "Мои оценки" nav badge in AppShell shows the ongoing count.
 */
export function ActivePeriodNotifier() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const { notification } = App.useApp();

  const { data: periods = [] } = useQuery({
    queryKey: ['my-assessment-periods'],
    queryFn: listMyAssessmentPeriods,
    enabled: !!user,
  });

  const active = periods.filter(p => p.is_active);
  const activeCount = active.length;

  useEffect(() => {
    if (!user || activeCount === 0) return;
    const key = `${SESSION_KEY_PREFIX}${user.id}`;
    if (sessionStorage.getItem(key)) return;
    sessionStorage.setItem(key, '1');

    const lead = activeCount === 1
      ? `Идёт период «${active[0].title}»`
      : `Активных периодов оценки: ${activeCount}`;

    notification.open({
      key: 'active-periods',  // dedupe in case StrictMode double-fires
      message: lead,
      description: 'Вы можете выставить оценки своим сотрудникам.',
      icon: <BellOutlined style={{ color: '#1F5EFF' }} />,
      btn: (
        <Button
          type="primary"
          size="small"
          onClick={() => {
            notification.destroy('active-periods');
            navigate('/assessments');
          }}
        >
          Перейти
        </Button>
      ),
      duration: 5,
      placement: 'topRight',
    });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id, activeCount]);

  return null;
}
