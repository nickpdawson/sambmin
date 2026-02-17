import { Result } from 'antd';
import { CrownOutlined } from '@ant-design/icons';

export default function FSMO() {
  return (
    <Result
      icon={<CrownOutlined />}
      title="FSMO Roles"
      subTitle="Manage Flexible Single Master Operations roles — coming in Phase 4"
    />
  );
}
