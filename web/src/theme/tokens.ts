import type { ThemeConfig } from 'antd';

export const lightTheme: ThemeConfig = {
  token: {
    colorPrimary: '#2563EB',
    colorInfo: '#2563EB',
    colorSuccess: '#16A34A',
    colorWarning: '#D97706',
    colorError: '#DC2626',
    borderRadius: 6,
    fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    fontFamilyCode: '"JetBrains Mono", "Fira Code", "Cascadia Code", monospace',
    fontSize: 14,
    controlHeight: 36,
    colorBgContainer: '#FFFFFF',
    colorBgLayout: '#F8FAFC',
    colorBgElevated: '#FFFFFF',
    colorBorder: '#E2E8F0',
    colorBorderSecondary: '#F1F5F9',
    colorText: '#0F172A',
    colorTextSecondary: '#64748B',
    colorTextTertiary: '#94A3B8',
    wireframe: false,
  },
  components: {
    Layout: {
      siderBg: '#FFFFFF',
      headerBg: '#FFFFFF',
      bodyBg: '#F8FAFC',
    },
    Menu: {
      itemBg: 'transparent',
      itemSelectedBg: '#EFF6FF',
      itemSelectedColor: '#2563EB',
      itemHoverBg: '#F8FAFC',
    },
    Table: {
      headerBg: '#F8FAFC',
      rowHoverBg: '#F1F5F9',
      borderColor: '#E2E8F0',
    },
    Card: {
      paddingLG: 20,
    },
  },
};

export const darkTheme: ThemeConfig = {
  token: {
    colorPrimary: '#60A5FA',
    colorInfo: '#60A5FA',
    colorSuccess: '#4ADE80',
    colorWarning: '#FBBF24',
    colorError: '#F87171',
    borderRadius: 6,
    fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    fontFamilyCode: '"JetBrains Mono", "Fira Code", "Cascadia Code", monospace',
    fontSize: 14,
    controlHeight: 36,
    colorBgContainer: '#1E293B',
    colorBgLayout: '#0F172A',
    colorBgElevated: '#1E293B',
    colorBorder: '#334155',
    colorBorderSecondary: '#1E293B',
    colorText: '#F1F5F9',
    colorTextSecondary: '#94A3B8',
    colorTextTertiary: '#64748B',
    wireframe: false,
  },
  components: {
    Layout: {
      siderBg: '#0F172A',
      headerBg: '#0F172A',
      bodyBg: '#0F172A',
    },
    Menu: {
      itemBg: 'transparent',
      itemSelectedBg: '#1E3A5F',
      itemSelectedColor: '#60A5FA',
      itemHoverBg: '#1E293B',
    },
    Table: {
      headerBg: '#1E293B',
      rowHoverBg: '#334155',
      borderColor: '#334155',
    },
    Card: {
      paddingLG: 20,
    },
  },
};
