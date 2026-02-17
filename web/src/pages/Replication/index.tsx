import { Result } from 'antd';
import { SyncOutlined } from '@ant-design/icons';

export default function Replication() {
  return (
    <Result
      icon={<SyncOutlined />}
      title="Replication"
      subTitle="Monitor replication topology and health — coming in Phase 3"
    />
  );
}
