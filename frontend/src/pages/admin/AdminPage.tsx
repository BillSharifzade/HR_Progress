import { useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Empty,
  Input,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
  theme as antdTheme,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CopyOutlined,
  DeleteOutlined,
  KeyOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';

import { PageHeader } from '../../components/PageHeader';
import { PageSkeleton } from '../../components/PageSkeleton';
import {
  getWorker,
  grantRole,
  listSections,
  listWorkerRoles,
  listWorkers,
  resetWorkerCredentials,
  revokeRole,
  type GrantRolePayload,
} from '../../api/workers';
import { listAllDepartments } from '../../api/competency';
import type {
  Department,
  RoleAssignment,
  Section,
  UserRole,
  WorkerSummary,
} from '../../types';
import { UserRoleLabel } from '../../types';

const { Text } = Typography;

const ROLES: UserRole[] = [
  'HR_ADMIN',
  'DEPT_HEAD',
  'SECTION_HEAD',
  'ASSESSOR',
  'PRECEPTOR',
  'ATS',
  'BOOK_SPACE',
];

function roleNeedsDept(role: UserRole): boolean {
  return role === 'DEPT_HEAD';
}

function roleNeedsSection(role: UserRole): boolean {
  return role === 'SECTION_HEAD';
}

const ROLE_COLOR: Record<UserRole, string> = {
  HR_ADMIN:     'magenta',
  DEPT_HEAD:    'geekblue',
  SECTION_HEAD: 'blue',
  ASSESSOR:     'gold',
  PRECEPTOR:    'green',
  ATS:          'cyan',
  BOOK_SPACE:   'purple',
};

interface GrantFormState {
  role: UserRole | null;
  departmentId: string | null;
  sectionId: string | null;
}

const emptyGrant: GrantFormState = { role: null, departmentId: null, sectionId: null };

export function AdminPage() {
  const { token } = antdTheme.useToken();
  const [search, setSearch] = useState('');
  const [selectedWorkerId, setSelectedWorkerId] = useState<string | null>(null);
  const [grant, setGrant] = useState<GrantFormState>(emptyGrant);
  const [messageApi, contextHolder] = message.useMessage();
  const qc = useQueryClient();

  const [credentialsOpen, setCredentialsOpen] = useState(false);
  const [issuedCredentials, setIssuedCredentials] = useState<{
    username: string;
    password: string;
  } | null>(null);

  const { data: workers = [], isLoading: workersLoading } = useQuery({
    queryKey: ['workers', { include_inactive: false }],
    queryFn: () => listWorkers(),
  });

  const { data: workerDetail } = useQuery({
    queryKey: ['worker', selectedWorkerId],
    queryFn: () => getWorker(selectedWorkerId!),
    enabled: !!selectedWorkerId,
  });

  const { data: departments = [] } = useQuery<Department[]>({
    queryKey: ['departments', 'all'],
    queryFn: listAllDepartments,
  });

  const { data: sectionsForGrant = [] } = useQuery<Section[]>({
    queryKey: ['sections', grant.departmentId],
    queryFn: () => listSections(grant.departmentId ?? undefined),
    enabled: !!grant.departmentId,
  });

  const { data: roles = [], isLoading: rolesLoading } = useQuery({
    queryKey: ['worker-roles', selectedWorkerId],
    queryFn: () => listWorkerRoles(selectedWorkerId!),
    enabled: !!selectedWorkerId,
  });

  const filteredWorkers = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return workers;
    return workers.filter(w =>
      w.full_name.toLowerCase().includes(q) ||
      (w.personnel_number ?? '').toLowerCase().includes(q) ||
      (w.department_name ?? '').toLowerCase().includes(q),
    );
  }, [workers, search]);

  const selectedWorker = workers.find(w => w.id === selectedWorkerId) ?? null;

  const grantMut = useMutation({
    mutationFn: (payload: GrantRolePayload) => grantRole(selectedWorkerId!, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['worker-roles', selectedWorkerId] });
      setGrant(emptyGrant);
      messageApi.success('Роль выдана');
    },
    onError: (err: { response?: { data?: { error?: { code?: string } } } }) => {
      const code = err?.response?.data?.error?.code;
      if (code === 'ROLE_EXISTS') {
        messageApi.warning('Эта роль с такой областью уже выдана');
      } else if (code === 'INVALID_SCOPE') {
        messageApi.error('Неверная область для роли');
      } else {
        messageApi.error('Не удалось выдать роль');
      }
    },
  });

  const revokeMut = useMutation({
    mutationFn: (assignmentId: string) => revokeRole(selectedWorkerId!, assignmentId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['worker-roles', selectedWorkerId] });
      messageApi.success('Роль отозвана');
    },
    onError: () => messageApi.error('Не удалось отозвать роль'),
  });

  const resetMut = useMutation({
    mutationFn: () => resetWorkerCredentials(selectedWorkerId!),
    onSuccess: (res) => {
      setIssuedCredentials(res);
      messageApi.success('Пароль сброшен');
    },
    onError: (err: { response?: { data?: { error?: { message?: string } } } }) => {
      messageApi.error(err?.response?.data?.error?.message ?? 'Не удалось сбросить пароль');
    },
  });

  const closeCredentialsModal = () => {
    setCredentialsOpen(false);
    setIssuedCredentials(null);
  };

  const copyToClipboard = async (text: string, label: string) => {
    try {
      await navigator.clipboard.writeText(text);
      messageApi.success(`${label} скопирован`);
    } catch {
      messageApi.warning('Не удалось скопировать');
    }
  };

  const handleGrant = () => {
    if (!grant.role || !selectedWorkerId) return;
    const payload: GrantRolePayload = { role: grant.role };
    if (roleNeedsDept(grant.role)) {
      if (!grant.departmentId) {
        messageApi.warning('Выберите департамент');
        return;
      }
      payload.scope_department_id = grant.departmentId;
    }
    if (roleNeedsSection(grant.role)) {
      if (!grant.sectionId) {
        messageApi.warning('Выберите отдел');
        return;
      }
      payload.scope_section_id = grant.sectionId;
    }
    grantMut.mutate(payload);
  };

  const workerColumns: ColumnsType<WorkerSummary> = [
    {
      title: 'ФИО',
      dataIndex: 'full_name',
      render: (name: string, row) => (
        <div>
          <Text strong>{name}</Text>
          {row.personnel_number && (
            <Text type="secondary" style={{ fontSize: 11, marginLeft: 8 }}>
              #{row.personnel_number}
            </Text>
          )}
        </div>
      ),
    },
    {
      title: 'Департамент',
      dataIndex: 'department_name',
      render: (v?: string | null) => v ?? <Text type="secondary">—</Text>,
    },
    {
      title: 'Грейд',
      dataIndex: 'grade_name',
      width: 180,
      render: (v?: string | null) => v ?? <Text type="secondary">—</Text>,
    },
  ];

  const roleColumns: ColumnsType<RoleAssignment> = [
    {
      title: 'Роль',
      dataIndex: 'role',
      render: (role: UserRole) => (
        <Tag color={ROLE_COLOR[role]}>{UserRoleLabel[role]}</Tag>
      ),
    },
    {
      title: 'Область',
      render: (_, row) => {
        const parts: string[] = [];
        if (row.scope_department) parts.push(row.scope_department);
        if (row.scope_section) parts.push(row.scope_section);
        return parts.length ? parts.join(' / ') : <Text type="secondary">—</Text>;
      },
    },
    {
      title: 'Выдал',
      width: 220,
      render: (_, row) => (
        <div style={{ fontSize: 12, lineHeight: 1.3 }}>
          <div>{row.granted_by_name ?? <Text type="secondary">система</Text>}</div>
          <Text type="secondary" style={{ fontSize: 11 }}>
            {dayjs(row.granted_at).format('DD.MM.YYYY HH:mm')}
          </Text>
        </div>
      ),
    },
    {
      title: '',
      width: 56,
      align: 'right',
      render: (_, row) => (
        <Popconfirm
          title="Отозвать роль?"
          okText="Отозвать"
          cancelText="Отмена"
          okButtonProps={{ danger: true }}
          onConfirm={() => revokeMut.mutate(row.id)}
        >
          <Button type="text" danger size="small" icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  if (workersLoading) return <PageSkeleton type="list" />;

  return (
    <>
      {contextHolder}
      <PageHeader
        title="Администрирование"
        subtitle="Управление ролями пользователей"
      />

      <Row gutter={16}>
        <Col xs={24} lg={11}>
          <Card
            size="small"
            title="Сотрудники"
            extra={
              <Input
                placeholder="Поиск"
                prefix={<SearchOutlined />}
                allowClear
                size="small"
                style={{ width: 200 }}
                value={search}
                onChange={e => setSearch(e.target.value)}
              />
            }
          >
            <Table
              dataSource={filteredWorkers}
              columns={workerColumns}
              rowKey="id"
              size="small"
              pagination={{ pageSize: 15, size: 'small' }}
              onRow={row => ({
                onClick: () => {
                  setSelectedWorkerId(row.id);
                  setGrant(emptyGrant);
                },
                style: {
                  cursor: 'pointer',
                  background: row.id === selectedWorkerId ? token.colorPrimaryBg : undefined,
                },
              })}
            />
          </Card>
        </Col>

        <Col xs={24} lg={13}>
          <Card
            size="small"
            title={
              selectedWorker
                ? <span>Роли: <Text strong>{selectedWorker.full_name}</Text></span>
                : 'Роли сотрудника'
            }
          >
            {!selectedWorkerId ? (
              <Empty description="Выберите сотрудника слева" />
            ) : (
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Card size="small" type="inner" title={<><KeyOutlined /> Учётные данные</>}>
                  <Space direction="vertical" size={4} style={{ width: '100%' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Text type="secondary" style={{ fontSize: 12, minWidth: 64 }}>Логин:</Text>
                      <Text code style={{ fontSize: 13 }}>{workerDetail?.username ?? '…'}</Text>
                      {workerDetail?.username && (
                        <Tooltip title="Скопировать логин">
                          <Button
                            type="text"
                            size="small"
                            icon={<CopyOutlined />}
                            onClick={() => copyToClipboard(workerDetail.username, 'Логин')}
                          />
                        </Tooltip>
                      )}
                    </div>
                    <Button
                      icon={<ReloadOutlined />}
                      onClick={() => {
                        setIssuedCredentials(null);
                        setCredentialsOpen(true);
                      }}
                    >
                      Сбросить пароль
                    </Button>
                  </Space>
                </Card>

                <Table
                  dataSource={roles}
                  columns={roleColumns}
                  rowKey="id"
                  size="small"
                  pagination={false}
                  loading={rolesLoading}
                  locale={{ emptyText: 'У сотрудника пока нет ролей' }}
                />

                <Card size="small" type="inner" title={<><PlusOutlined /> Выдать роль</>}>
                  <Space direction="vertical" size={8} style={{ width: '100%' }}>
                    <Select
                      placeholder="Роль"
                      style={{ width: '100%' }}
                      value={grant.role ?? undefined}
                      onChange={(role: UserRole) =>
                        setGrant({ role, departmentId: null, sectionId: null })
                      }
                      options={ROLES.map(r => ({ value: r, label: UserRoleLabel[r] }))}
                    />

                    {grant.role && (roleNeedsDept(grant.role) || roleNeedsSection(grant.role)) && (
                      <Select
                        placeholder="Департамент"
                        style={{ width: '100%' }}
                        value={grant.departmentId ?? undefined}
                        onChange={(departmentId: string) =>
                          setGrant(g => ({ ...g, departmentId, sectionId: null }))
                        }
                        options={departments.map(d => ({
                          value: d.id,
                          label: `${d.code} — ${d.name}`,
                        }))}
                      />
                    )}

                    {grant.role && roleNeedsSection(grant.role) && (
                      <Select
                        placeholder="Отдел"
                        style={{ width: '100%' }}
                        value={grant.sectionId ?? undefined}
                        disabled={!grant.departmentId}
                        onChange={(sectionId: string) =>
                          setGrant(g => ({ ...g, sectionId }))
                        }
                        options={sectionsForGrant.map(s => ({
                          value: s.id,
                          label: s.name,
                        }))}
                        notFoundContent="В этом департаменте нет отделов"
                      />
                    )}

                    {grant.role === 'HR_ADMIN' && (
                      <Alert
                        type="warning"
                        showIcon
                        message="Эта роль даёт полный доступ ко всей системе."
                      />
                    )}

                    <Button
                      type="primary"
                      icon={<PlusOutlined />}
                      block
                      disabled={!grant.role}
                      loading={grantMut.isPending}
                      onClick={handleGrant}
                    >
                      Выдать
                    </Button>
                  </Space>
                </Card>
              </Space>
            )}
          </Card>
        </Col>
      </Row>

      <Modal
        title="Сброс пароля"
        open={credentialsOpen}
        onCancel={closeCredentialsModal}
        destroyOnClose
        footer={
          issuedCredentials ? (
            <Button type="primary" onClick={closeCredentialsModal}>Готово</Button>
          ) : (
            <Space>
              <Button onClick={closeCredentialsModal}>Отмена</Button>
              <Button
                danger
                type="primary"
                loading={resetMut.isPending}
                onClick={() => resetMut.mutate()}
              >
                Сгенерировать новый пароль
              </Button>
            </Space>
          )
        }
      >
        {issuedCredentials ? (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Alert
              type="warning"
              showIcon
              message="Передайте эти данные сотруднику"
              description="Пароль больше не будет показан. Сотрудник не сможет сменить его самостоятельно — только администратор."
            />
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text type="secondary" style={{ minWidth: 72 }}>Логин:</Text>
              <Text code copyable={{ text: issuedCredentials.username }}>
                {issuedCredentials.username}
              </Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Text type="secondary" style={{ minWidth: 72 }}>Пароль:</Text>
              <Text code copyable={{ text: issuedCredentials.password }}>
                {issuedCredentials.password}
              </Text>
            </div>
          </Space>
        ) : (
          <Text type="secondary" style={{ fontSize: 13 }}>
            Будет сгенерирован новый случайный пароль (16 символов).
            Прежний пароль перестанет работать сразу после нажатия кнопки.
          </Text>
        )}
      </Modal>
    </>
  );
}
