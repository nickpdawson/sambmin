import { Result } from 'antd';
import { ClusterOutlined } from '@ant-design/icons';

export default function Sites() {
  return (
    <Result
      icon={<ClusterOutlined />}
      title="Sites & Services"
      subTitle="Manage AD sites, subnets, and site links — coming in Phase 3"
    />
  );
}
