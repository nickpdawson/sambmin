import { Result } from 'antd';
import { TeamOutlined } from '@ant-design/icons';

export default function Groups() {
  return (
    <Result
      icon={<TeamOutlined />}
      title="Groups"
      subTitle="Manage security and distribution groups — coming in Phase 2"
    />
  );
}
