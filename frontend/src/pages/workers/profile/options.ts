/**
 * Canonical answer options for the digital-profile questionnaire.
 *
 * These are the values the form itself offered, taken from the 146 responses
 * rather than invented: every list below covers the real answers, and the
 * multi-selects additionally saw a handful of free-text additions, which is why
 * their controls stay in `tags` mode instead of being closed dropdowns.
 *
 * Selects that reflect a closed question use `allowCustom: false`; a stored
 * value outside the list is still displayed, never silently dropped.
 */

export const EDUCATION_LEVELS = [
  'Высшее',
  'Магистратура',
  'Неоконченное высшее',
  'Среднее специальное',
  'Кандидат наук / PhD',
  'Бакалавриат неоконченный',
];

export const EXPERIENCE_BANDS = [
  'До 1 года',
  '1–3 года',
  '3–5 лет',
  '5–10 лет',
  'Более 10 лет',
];

export const CAREER_GOALS = [
  'Развиваться как эксперт в своем направлении',
  'Развиваться в управлении проектами',
  'Стать Руководителем Отдела',
  'Стать Руководителем Департамента',
  'Освоить новую профессиональную область',
  'Пока не определился',
];

export const MOBILITY_OPTIONS = [
  'да',
  'нет',
  'только внутри текущего направления деятельности',
];

export const RELOCATION_OPTIONS = ['да', 'нет', 'по согласованию'];

export const INTERNAL_PROJECT_OPTIONS = ['да', 'нет', 'по возможности'];

export const TEACHING_OPTIONS = ['да', 'нет', 'в будущем'];

export const LEARNING_HOURS_BANDS = [
  'До 2 часов',
  '2–4 часа',
  '4–8 часов',
  'Более 8 часов',
];

export const DEVELOPMENT_DIRECTIONS = [
  'Управление проектами',
  'Финансы',
  'Аналитика',
  'Автоматизация процессов',
  'Управление персоналом',
  'Информационные технологии',
  'Продажи',
  'Тренинги',
  'Производство',
];

export const PROFESSIONAL_INTERESTS = [
  'Управление проектами',
  'Психология',
  'Лидерство',
  'Финансы',
  'Аналитика данных',
  'Автоматизация процессов',
  'Искусственный интеллект',
  'Коммуникации',
  'Управление людьми',
  'Маркетинг',
  'Продажи',
  'Производство',
];

export const LEARNING_FORMATS = [
  'Практические задания',
  'Очные тренинги',
  'Наставничество',
  'Самостоятельное изучение литературы',
  'Онлайн-курсы',
  'Видеоуроки',
  'Вебинары',
];

/** Languages the form asked about, in the order it listed them. */
export const KNOWN_LANGUAGES = [
  'Таджикский',
  'Русский',
  'Английский',
  'Китайский',
  'Немецкий',
  'Турецкий',
];

/**
 * Colour per CEFR band — warm for beginner, green for fluent — so a row of
 * language chips is scannable without reading the levels.
 */
export const CEFR_COLORS: Record<string, string> = {
  C2: 'green',
  C1: 'green',
  B2: 'blue',
  B1: 'cyan',
  A2: 'orange',
  A1: 'default',
};

/** Builds Select options, keeping any stored value that is not in the list. */
export function withCurrent(list: string[], current?: string | null): { value: string; label: string }[] {
  const values = current && !list.includes(current) ? [current, ...list] : list;
  return values.map((v) => ({ value: v, label: v }));
}
