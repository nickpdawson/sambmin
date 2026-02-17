import { Result } from 'antd';
import { FileProtectOutlined } from '@ant-design/icons';

export default function GPO() {
  return (
    <Result
      icon={<FileProtectOutlined />}
      title="Group Policy"
      subTitle="Manage Group Policy Objects — coming in Phase 4"
    />
  );
}
