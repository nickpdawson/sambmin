import { Result } from 'antd';
import { UserOutlined } from '@ant-design/icons';

export default function Users() {
  return (
    <Result
      icon={<UserOutlined />}
      title="Users"
      subTitle="Manage domain user accounts — coming in Phase 2"
    />
  );
}
