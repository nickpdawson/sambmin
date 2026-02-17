import { Result } from 'antd';
import { ApartmentOutlined } from '@ant-design/icons';

export default function OUs() {
  return (
    <Result
      icon={<ApartmentOutlined />}
      title="Organizational Units"
      subTitle="Manage OU hierarchy — coming in Phase 2"
    />
  );
}
