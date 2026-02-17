import { Result } from 'antd';
import { KeyOutlined } from '@ant-design/icons';

export default function Kerberos() {
  return (
    <Result
      icon={<KeyOutlined />}
      title="Kerberos"
      subTitle="Manage Kerberos principals and keytabs — coming in Phase 4"
    />
  );
}
