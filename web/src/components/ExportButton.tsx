import { Dropdown, Button } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import type { MenuProps } from 'antd';

interface ExportButtonProps {
  data: Record<string, unknown>[];
  filename: string;
  columns?: { key: string; title: string }[];
}

function toCSV(data: Record<string, unknown>[], columns?: { key: string; title: string }[]): string {
  if (data.length === 0) return '';

  const keys = columns
    ? columns.map((c) => c.key)
    : Object.keys(data[0]);
  const headers = columns
    ? columns.map((c) => c.title)
    : keys;

  const escape = (val: unknown): string => {
    const s = val == null ? '' : Array.isArray(val) ? val.join('; ') : String(val);
    if (s.includes(',') || s.includes('"') || s.includes('\n')) {
      return `"${s.replace(/"/g, '""')}"`;
    }
    return s;
  };

  const rows = data.map((row) => keys.map((k) => escape(row[k])).join(','));
  return [headers.map((h) => escape(h)).join(','), ...rows].join('\n');
}

function download(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export default function ExportButton({ data, filename, columns }: ExportButtonProps) {
  const items: MenuProps['items'] = [
    {
      key: 'csv',
      label: 'Export as CSV',
      onClick: () => {
        const csv = toCSV(data, columns);
        download(csv, `${filename}.csv`, 'text/csv;charset=utf-8');
      },
    },
    {
      key: 'json',
      label: 'Export as JSON',
      onClick: () => {
        const json = JSON.stringify(data, null, 2);
        download(json, `${filename}.json`, 'application/json');
      },
    },
  ];

  return (
    <Dropdown menu={{ items }} trigger={['click']} disabled={data.length === 0}>
      <Button icon={<DownloadOutlined />} disabled={data.length === 0}>
        Export
      </Button>
    </Dropdown>
  );
}
