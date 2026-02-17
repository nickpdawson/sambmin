import { Result } from 'antd';
import { GlobalOutlined } from '@ant-design/icons';

export default function DNS() {
  return (
    <Result
      icon={<GlobalOutlined />}
      title="DNS Management"
      subTitle="Manage DNS zones and records — coming in Phase 3"
    />
  );
}
