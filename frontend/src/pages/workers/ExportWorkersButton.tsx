import { Button, Tooltip, message } from 'antd';
import { FileExcelOutlined } from '@ant-design/icons';
import { useMutation } from '@tanstack/react-query';

import { exportWorkers, readBlobError, type ListWorkersParams } from '../../api/workers';

/**
 * Downloads the register as a formatted XLSX workbook.
 *
 * The button passes the filters the list is currently showing, so what lands in
 * the file is what the user is looking at — with no filters set that is simply
 * every worker they may see. The tooltip says which of the two is happening,
 * because "Экспорт" on a filtered screen is otherwise ambiguous.
 */
export function ExportWorkersButton({
  params,
  disabled,
}: {
  params: ListWorkersParams;
  disabled?: boolean;
}) {
  const [messageApi, contextHolder] = message.useMessage();

  const mut = useMutation({
    mutationFn: () => exportWorkers(params),
    onSuccess: (fileName) => messageApi.success(`Файл «${fileName}» сохранён`),
    onError: async (err) => {
      messageApi.error(
        (await readBlobError(err)) ?? 'Не удалось сформировать файл выгрузки',
      );
    },
  });

  const filtered = Object.values(params).some((v) => v !== undefined && v !== '');

  return (
    <>
      {contextHolder}
      <Tooltip
        title={
          filtered
            ? 'Выгрузка в Excel по текущим фильтрам: полный профиль каждого сотрудника на отдельных листах'
            : 'Выгрузка всех сотрудников в Excel: полный профиль каждого на отдельных листах'
        }
      >
        <Button
          icon={<FileExcelOutlined />}
          loading={mut.isPending}
          disabled={disabled}
          onClick={() => mut.mutate()}
        >
          Экспорт в Excel
        </Button>
      </Tooltip>
    </>
  );
}
