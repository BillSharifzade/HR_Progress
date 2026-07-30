import { useMemo } from 'react';
import { Col, Row, Space, Tag, Typography } from 'antd';
import dayjs from 'dayjs';

import {
  CampaignStatusColor, CampaignStatusLabel, type CampaignStatus,
} from '../../types';
import type { DashboardOverview } from '../../api/dashboard';
import { StatTile } from '../../components/charts/StatTile';
import { ChartCard } from '../../components/charts/ChartCard';
import {
  BarChart, ColumnChart, DivergingBar, DivergingLegend, Meter, TrendArea,
} from '../../components/charts/Charts';

const { Text } = Typography;

/** Competencies shown in the gap chart — enough to see both tails without
 *  turning it into a wall of bars. */
const GAP_LIMIT = 10;

/**
 * The analytics body of the dashboard. Pure presentation: it takes a loaded
 * overview and renders it, which keeps it renderable in isolation.
 */
export function DashboardAnalytics({ data }: { data: DashboardOverview }) {
  const gaps = useMemo(
    () => data.competency_gaps.slice(0, GAP_LIMIT).map(g => ({
      label: g.name,
      value: Number(g.delta.toFixed(2)),
      actual: g.avg_score,
      target: g.required_min,
      meta: `Оценок: ${g.assessed}`,
    })),
    [data],
  );

  const trend = useMemo(
    () => data.hiring_trend.map(p => ({
      label: dayjs(`${p.month}-01`).format('MMM YY'),
      value: p.total,
      note: p.hired > 0 ? `Принято за месяц: ${p.hired}` : 'Приёмов не было',
    })),
    [data],
  );

  const k = data.kpis;
  const coveragePct = k.active_workers > 0
    ? Math.round((k.assessed_workers / k.active_workers) * 100)
    : 0;

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {/* ── Headline numbers ───────────────────────────────────────────── */}
      <Row gutter={[16, 16]}>
        <Col xs={12} md={8} xl={4}>
          <StatTile title="Сотрудников" value={k.active_workers} accent />
        </Col>
        <Col xs={12} md={8} xl={4}>
          <StatTile title="Департаментов" value={k.departments} delay={60} />
        </Col>
        <Col xs={12} md={8} xl={4}>
          <StatTile title="Компетенций" value={k.competencies} delay={120} />
        </Col>
        <Col xs={12} md={8} xl={4}>
          <StatTile title="Активных кампаний" value={k.active_campaigns} delay={180} />
        </Col>
        <Col xs={12} md={8} xl={4}>
          <StatTile
            title="Оценено сотрудников"
            value={k.assessed_workers}
            hint={`${coveragePct}% от штата`}
            delay={240}
          />
        </Col>
        <Col xs={12} md={8} xl={4}>
          <StatTile
            title="Средний балл"
            value={k.avg_score}
            suffix={k.avg_score !== null ? '/ 10' : undefined}
            delay={300}
          />
        </Col>
      </Row>

      {/* ── Org shape ──────────────────────────────────────────────────── */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <ChartCard
            title="Сотрудники по департаментам"
            subtitle="Активные сотрудники"
            rowKey="label"
            tableData={data.headcount_by_department}
            tableColumns={[
              { title: 'Департамент', dataIndex: 'label' },
              { title: 'Сотрудников', dataIndex: 'value', width: 120, align: 'right' },
            ]}
          >
            <BarChart data={data.headcount_by_department} unit="чел." />
          </ChartCard>
        </Col>
        <Col xs={24} lg={12}>
          <ChartCard
            title="Сотрудники по грейдам"
            subtitle="От стажёра к руководителю департамента"
            rowKey="label"
            tableData={data.headcount_by_grade}
            tableColumns={[
              { title: 'Грейд', dataIndex: 'label' },
              { title: 'Сотрудников', dataIndex: 'value', width: 120, align: 'right' },
            ]}
          >
            {/* Horizontal: grade names are long enough that column labels
                would collide, and the ladder order carries the ordinal ramp. */}
            <BarChart data={data.headcount_by_grade} unit="чел." ordinal />
          </ChartCard>
        </Col>
      </Row>

      {/* ── Competency health ──────────────────────────────────────────── */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={14}>
          <ChartCard
            title="Компетенции: факт против требования"
            subtitle={
              data.competency_gaps.length > GAP_LIMIT
                ? `${GAP_LIMIT} из ${data.competency_gaps.length}, начиная с наибольшего дефицита`
                : 'Отклонение среднего балла от требуемого минимума'
            }
            extra={<DivergingLegend />}
            rowKey="name"
            emptyText="Нет консолидированных оценок"
            tableData={data.competency_gaps}
            tableColumns={[
              { title: 'Компетенция', dataIndex: 'name' },
              { title: 'Вид', dataIndex: 'kind', width: 70 },
              { title: 'Средний', dataIndex: 'avg_score', width: 90, align: 'right', render: (v: number) => v.toFixed(2) },
              { title: 'Требуется', dataIndex: 'required_min', width: 100, align: 'right', render: (v: number) => v.toFixed(1) },
              {
                title: 'Отклонение', dataIndex: 'delta', width: 110, align: 'right',
                render: (v: number) => (
                  <Text type={v < 0 ? 'danger' : 'success'}>{v > 0 ? '+' : ''}{v.toFixed(2)}</Text>
                ),
              },
            ]}
          >
            <DivergingBar data={gaps} />
          </ChartCard>
        </Col>
        <Col xs={24} lg={10}>
          <ChartCard
            title="Распределение оценок"
            subtitle="Консолидированные баллы по шкале 0–10"
            rowKey="label"
            emptyText="Нет консолидированных оценок"
            tableData={data.score_distribution}
            tableColumns={[
              { title: 'Диапазон', dataIndex: 'label' },
              { title: 'Оценок', dataIndex: 'value', width: 100, align: 'right' },
            ]}
          >
            <ColumnChart data={data.score_distribution} ordinal unit="оценок" height={252} />
          </ChartCard>
        </Col>
      </Row>

      {/* ── Campaigns & growth ─────────────────────────────────────────── */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <ChartCard
            title="Ход кампаний"
            subtitle="Заполнено ячеек оценки"
            rowKey="period_id"
            emptyText="Нет активных кампаний"
            tableData={data.campaign_progress}
            tableColumns={[
              { title: 'Кампания', dataIndex: 'title' },
              { title: 'Участников', dataIndex: 'assessees', width: 110, align: 'right' },
              { title: 'Готово', dataIndex: 'done', width: 90, align: 'right' },
              { title: 'Всего', dataIndex: 'expected', width: 90, align: 'right' },
            ]}
          >
            <div style={{ paddingTop: 4 }}>
              {data.campaign_progress.map((c, i) => (
                <Meter
                  key={c.period_id}
                  delay={i * 70}
                  label={
                    <Space size={6}>
                      <span>{c.title}</span>
                      <Tag
                        color={CampaignStatusColor[c.status as CampaignStatus]}
                        style={{ marginInlineEnd: 0 }}
                      >
                        {CampaignStatusLabel[c.status as CampaignStatus] ?? c.status}
                      </Tag>
                    </Space>
                  }
                  value={c.done}
                  max={c.expected}
                  caption={
                    c.expected > 0
                      ? `${c.done} / ${c.expected} · ${Math.round((c.done / c.expected) * 100)}%`
                      : 'не настроена'
                  }
                />
              ))}
            </div>
          </ChartCard>
        </Col>
        <Col xs={24} lg={12}>
          <ChartCard
            title="Численность"
            subtitle="Накопительно, за последние 12 месяцев"
            rowKey="month"
            tableData={data.hiring_trend}
            tableColumns={[
              { title: 'Месяц', dataIndex: 'month' },
              { title: 'Принято', dataIndex: 'hired', width: 100, align: 'right' },
              { title: 'Всего', dataIndex: 'total', width: 100, align: 'right' },
            ]}
          >
            <TrendArea data={trend} unit="чел." height={252} />
          </ChartCard>
        </Col>
      </Row>
    </Space>
  );
}
