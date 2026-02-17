import { Result } from 'antd';
import { SettingOutlined } from '@ant-design/icons';

export default function Settings() {
  return (
    <Result
      icon={<SettingOutlined />}
      title="Settings"
      subTitle="Configure Sambmin — coming in Phase 2"
    />
  );
}
