import { Result } from 'antd';
import { DatabaseOutlined } from '@ant-design/icons';

export default function Schema() {
  return (
    <Result
      icon={<DatabaseOutlined />}
      title="Schema"
      subTitle="Browse and manage AD schema — coming in Phase 5"
    />
  );
}
