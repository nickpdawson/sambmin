import { Result } from 'antd';
import { DesktopOutlined } from '@ant-design/icons';

export default function Computers() {
  return (
    <Result
      icon={<DesktopOutlined />}
      title="Computers"
      subTitle="Manage computer and machine accounts — coming in Phase 2"
    />
  );
}
