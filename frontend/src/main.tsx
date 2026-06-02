import React from 'react';
import ReactDOM from 'react-dom/client';
import { ConfigProvider, theme as antdTheme } from 'antd';
import ruRUBase from 'antd/locale/ru_RU';

const RU_SHORT_MONTHS = [
  'Янв.', 'Февр.', 'Март', 'Апр.', 'Май', 'Июнь',
  'Июль', 'Авг.', 'Сент.', 'Окт.', 'Нояб.', 'Дек.',
];

const ruRU = {
  ...ruRUBase,
  DatePicker: {
    ...ruRUBase.DatePicker!,
    lang: {
      ...ruRUBase.DatePicker!.lang,
      shortMonths: RU_SHORT_MONTHS,
    },
  },
  Calendar: {
    ...ruRUBase.Calendar!,
    lang: {
      ...ruRUBase.Calendar!.lang,
      shortMonths: RU_SHORT_MONTHS,
    },
  },
};
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import dayjs from 'dayjs';
import 'dayjs/locale/ru';

import App from './App';
import { baseTokens, darkTokens } from './theme';
import { AuthProvider } from './auth/AuthProvider';
import { ThemeModeProvider, useThemeMode } from './theme/ThemeContext';

dayjs.locale('ru');

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000, refetchOnWindowFocus: false },
  },
});

function ThemedApp() {
  const { mode } = useThemeMode();
  const isDark = mode === 'dark';
  return (
    <ConfigProvider
      locale={ruRU}
      theme={{
        token: isDark ? darkTokens : baseTokens,
        algorithm: isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
      }}
    >
      <App />
    </ConfigProvider>
  );
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ThemeModeProvider>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <AuthProvider>
            <ThemedApp />
          </AuthProvider>
        </BrowserRouter>
      </QueryClientProvider>
    </ThemeModeProvider>
  </React.StrictMode>,
);
