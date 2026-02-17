import { Result } from 'antd';
import { AuditOutlined } from '@ant-design/icons';

export default function AuditLog() {
  return (
    <Result
      icon={<AuditOutlined />}
      title="Audit Log"
      subTitle="View administrative action history — coming in Phase 2"
    />
  );
}
