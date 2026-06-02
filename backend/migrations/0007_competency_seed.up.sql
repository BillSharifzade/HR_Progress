-- Seed: 13 universal competencies (ЛК × 11, УК × 2).

INSERT INTO competencies (code, kind, name, description, why_important, sort_order) VALUES
    ('LK_COMM', 'LK', 'Коммуникация',
     'Способность ясно выражать мысли, внимательно слушать собеседника, задавать уточняющие вопросы и корректно доносить свою позицию.',
     'Без эффективной коммуникации возникают ошибки, недопонимание и потери времени. Чёткое взаимодействие ускоряет выполнение задач и повышает качество решений.',
     1),
    ('LK_ADAP', 'LK', 'Адаптивность',
     'Способность сохранять эффективность в условиях изменений, быстро перестраивать подходы и корректировать решения без потери результата.',
     'Без способности адаптироваться сотрудники сопротивляются изменениям и теряют эффективность в быстро меняющейся среде.',
     2),
    ('LK_PERS', 'LK', 'Настойчивость',
     'Способность сохранять фокус на цели и продолжать движение к результату несмотря на трудности, сопротивление и неудачи.',
     'Настойчивость отличает сотрудников, которые доводят задачи до результата, от тех, кто ограничивается формальным выполнением.',
     3),
    ('LK_RESP', 'LK', 'Ответственность',
     'Способность брать на себя обязательства за задачи, решения и результат своей работы, соблюдать договорённости и сроки, доводить задачи до результата.',
     'Без ответственности задачи формально выполняются, но результат не достигается. Срываются сроки, ошибки перекладываются на других.',
     4),
    ('LK_CREA', 'LK', 'Креативность',
     'Способность находить нестандартные решения и предлагать новые подходы при решении задач.',
     'В условиях конкуренции выигрывают сотрудники, которые способны находить новые решения и улучшать процессы.',
     5),
    ('LK_EI', 'LK', 'Эмоциональный интеллект',
     'Способность распознавать и управлять своими эмоциями, корректно воспринимать эмоции других людей и выстраивать взаимодействие с учётом этого.',
     'Без эмоционального интеллекта усиливаются конфликты, снижается доверие и ухудшается рабочая атмосфера.',
     6),
    ('LK_TEAM', 'LK', 'Командность',
     'Способность эффективно взаимодействовать с коллегами, учитывать общую цель команды и вносить вклад в общий результат.',
     'Без командной работы сотрудники действуют разрозненно, что снижает скорость и эффективность выполнения задач.',
     7),
    ('LK_DETA', 'LK', 'Внимательность к деталям',
     'Способность замечать важные нюансы и отклонения, работать аккуратно и доводить задачи до высокого уровня качества.',
     'Недостаточная внимательность приводит к ошибкам, доработкам и снижению качества работы.',
     8),
    ('LK_LOGI', 'LK', 'Логическое мышление',
     'Способность выстраивать причинно-следственные связи, аргументировать выводы и выявлять ошибки в рассуждениях.',
     'Позволяет принимать решения на основе фактов, видеть причинно-следственные связи и снижать риск ошибок.',
     9),
    ('LK_ANAL', 'LK', 'Аналитическое мышление',
     'Способность анализировать информацию и данные, выявлять закономерности, структурировать задачи и делать обоснованные выводы.',
     'Помогает выявлять реальные причины проблем, анализировать данные и принимать обоснованные решения.',
     10),
    ('LK_SELF', 'LK', 'Самостоятельность',
     'Способность самостоятельно организовывать работу, принимать решения в зоне ответственности и доводить задачи до результата без постоянного контроля.',
     'Повышает скорость работы команды и снижает операционную нагрузку на руководителей.',
     11),
    ('UK_DECI', 'UK', 'Навык принятия решений',
     'Способность своевременно выбирать и обосновывать оптимальный вариант действий в условиях ограниченного времени и неполной информации.',
     'Позволяет своевременно выбирать оптимальные варианты действий и снижает зависимость команды от постоянных согласований.',
     12),
    ('UK_LEAD', 'UK', 'Лидерские навыки',
     'Способность влиять на людей, задавать направление работы и объединять сотрудников для достижения общих целей.',
     'Позволяют удерживать фокус команды, задавать направление работы и объединять сотрудников для достижения общих результатов.',
     13);

-- Seed dept requirements for grades 1-4 (Стажёр→Главный Специалист) for all 7 departments.
-- Uses a DO block to resolve names to UUIDs cleanly.
DO $$
DECLARE
    -- competency IDs
    c_comm uuid; c_adap uuid; c_pers uuid; c_resp uuid; c_crea uuid;
    c_ei   uuid; c_team uuid; c_deta uuid; c_logi uuid; c_anal uuid;
    c_self uuid; c_deci uuid; c_lead uuid;
    -- grade IDs
    g1 uuid; g2 uuid; g3 uuid; g4 uuid;
    -- dept IDs
    d_fed uuid; d_dfp uuid; d_bud uuid; d_ahd uuid; d_dzl uuid; d_dit uuid; d_dcr uuid;
BEGIN
    SELECT id INTO c_comm FROM competencies WHERE code = 'LK_COMM';
    SELECT id INTO c_adap FROM competencies WHERE code = 'LK_ADAP';
    SELECT id INTO c_pers FROM competencies WHERE code = 'LK_PERS';
    SELECT id INTO c_resp FROM competencies WHERE code = 'LK_RESP';
    SELECT id INTO c_crea FROM competencies WHERE code = 'LK_CREA';
    SELECT id INTO c_ei   FROM competencies WHERE code = 'LK_EI';
    SELECT id INTO c_team FROM competencies WHERE code = 'LK_TEAM';
    SELECT id INTO c_deta FROM competencies WHERE code = 'LK_DETA';
    SELECT id INTO c_logi FROM competencies WHERE code = 'LK_LOGI';
    SELECT id INTO c_anal FROM competencies WHERE code = 'LK_ANAL';
    SELECT id INTO c_self FROM competencies WHERE code = 'LK_SELF';
    SELECT id INTO c_deci FROM competencies WHERE code = 'UK_DECI';
    SELECT id INTO c_lead FROM competencies WHERE code = 'UK_LEAD';

    SELECT id INTO g1 FROM grades WHERE level = 1;
    SELECT id INTO g2 FROM grades WHERE level = 2;
    SELECT id INTO g3 FROM grades WHERE level = 3;
    SELECT id INTO g4 FROM grades WHERE level = 4;

    SELECT id INTO d_fed FROM departments WHERE code = 'ФЭД';
    SELECT id INTO d_dfp FROM departments WHERE code = 'ДФП';
    SELECT id INTO d_bud FROM departments WHERE code = 'БЮД';
    SELECT id INTO d_ahd FROM departments WHERE code = 'АХД';
    SELECT id INTO d_dzl FROM departments WHERE code = 'ДЗЛ';
    SELECT id INTO d_dit FROM departments WHERE code = 'ДИТ';
    SELECT id INTO d_dcr FROM departments WHERE code = 'ДЧР';

    -- ФЭД
    INSERT INTO dept_competency_requirements (department_id, competency_id, grade_id, required_min, is_key) VALUES
        (d_fed, c_comm, g1, 3, false), (d_fed, c_comm, g2, 4, false), (d_fed, c_comm, g3, 6, true),  (d_fed, c_comm, g4, 7, true),
        (d_fed, c_adap, g1, 3, false), (d_fed, c_adap, g2, 4, false), (d_fed, c_adap, g3, 4, false), (d_fed, c_adap, g4, 6, false),
        (d_fed, c_pers, g1, 2, false), (d_fed, c_pers, g2, 3, false), (d_fed, c_pers, g3, 4, false), (d_fed, c_pers, g4, 5, false),
        (d_fed, c_resp, g1, 3, false), (d_fed, c_resp, g2, 4, false), (d_fed, c_resp, g3, 5, true),  (d_fed, c_resp, g4, 7, true),
        (d_fed, c_crea, g1, 2, false), (d_fed, c_crea, g2, 3, false), (d_fed, c_crea, g3, 4, false), (d_fed, c_crea, g4, 5, false),
        (d_fed, c_ei,   g1, 3, false), (d_fed, c_ei,   g2, 4, false), (d_fed, c_ei,   g3, 5, false), (d_fed, c_ei,   g4, 6, false),
        (d_fed, c_team, g1, 4, false), (d_fed, c_team, g2, 5, false), (d_fed, c_team, g3, 6, false), (d_fed, c_team, g4, 7, false),
        (d_fed, c_deta, g1, 4, true),  (d_fed, c_deta, g2, 4, true),  (d_fed, c_deta, g3, 5, true),  (d_fed, c_deta, g4, 6, true),
        (d_fed, c_logi, g1, 3, true),  (d_fed, c_logi, g2, 4, true),  (d_fed, c_logi, g3, 5, true),  (d_fed, c_logi, g4, 6, true),
        (d_fed, c_anal, g1, 3, true),  (d_fed, c_anal, g2, 4, true),  (d_fed, c_anal, g3, 5, true),  (d_fed, c_anal, g4, 7, true),
        (d_fed, c_self, g1, 2, false), (d_fed, c_self, g2, 3, false), (d_fed, c_self, g3, 4, true),  (d_fed, c_self, g4, 6, true),
        (d_fed, c_deci, g1, 2, false), (d_fed, c_deci, g2, 3, false), (d_fed, c_deci, g3, 5, false), (d_fed, c_deci, g4, 7, true),
        (d_fed, c_lead, g2, 3, false), (d_fed, c_lead, g3, 4, false), (d_fed, c_lead, g4, 6, true);

    -- ДФП
    INSERT INTO dept_competency_requirements (department_id, competency_id, grade_id, required_min, is_key) VALUES
        (d_dfp, c_comm, g1, 3, true),  (d_dfp, c_comm, g2, 4, true),  (d_dfp, c_comm, g3, 6, true),  (d_dfp, c_comm, g4, 7, true),
        (d_dfp, c_adap, g1, 3, false), (d_dfp, c_adap, g2, 4, false), (d_dfp, c_adap, g3, 5, false), (d_dfp, c_adap, g4, 6, false),
        (d_dfp, c_pers, g1, 2, false), (d_dfp, c_pers, g2, 3, false), (d_dfp, c_pers, g3, 4, false), (d_dfp, c_pers, g4, 6, false),
        (d_dfp, c_resp, g1, 3, false), (d_dfp, c_resp, g2, 4, false), (d_dfp, c_resp, g3, 5, true),  (d_dfp, c_resp, g4, 6, true),
        (d_dfp, c_crea, g1, 2, false), (d_dfp, c_crea, g2, 3, false), (d_dfp, c_crea, g3, 4, false), (d_dfp, c_crea, g4, 5, false),
        (d_dfp, c_ei,   g1, 3, false), (d_dfp, c_ei,   g2, 4, false), (d_dfp, c_ei,   g3, 5, true),  (d_dfp, c_ei,   g4, 6, true),
        (d_dfp, c_team, g1, 4, false), (d_dfp, c_team, g2, 5, false), (d_dfp, c_team, g3, 6, false), (d_dfp, c_team, g4, 7, false),
        (d_dfp, c_deta, g1, 4, false), (d_dfp, c_deta, g2, 4, false), (d_dfp, c_deta, g3, 5, true),  (d_dfp, c_deta, g4, 6, true),
        (d_dfp, c_logi, g1, 3, true),  (d_dfp, c_logi, g2, 4, true),  (d_dfp, c_logi, g3, 5, true),  (d_dfp, c_logi, g4, 6, true),
        (d_dfp, c_anal, g1, 3, true),  (d_dfp, c_anal, g2, 4, true),  (d_dfp, c_anal, g3, 6, true),  (d_dfp, c_anal, g4, 7, true),
        (d_dfp, c_self, g1, 2, false), (d_dfp, c_self, g2, 3, false), (d_dfp, c_self, g3, 4, false), (d_dfp, c_self, g4, 6, false),
        (d_dfp, c_deci, g1, 2, false), (d_dfp, c_deci, g2, 3, false), (d_dfp, c_deci, g3, 6, false), (d_dfp, c_deci, g4, 7, true),
        (d_dfp, c_lead, g2, 3, false), (d_dfp, c_lead, g3, 4, false), (d_dfp, c_lead, g4, 6, true);

    -- БЮД
    INSERT INTO dept_competency_requirements (department_id, competency_id, grade_id, required_min, is_key) VALUES
        (d_bud, c_comm, g1, 3, false), (d_bud, c_comm, g2, 3, false), (d_bud, c_comm, g3, 4, false), (d_bud, c_comm, g4, 5, false),
        (d_bud, c_adap, g1, 2, false), (d_bud, c_adap, g2, 3, false), (d_bud, c_adap, g3, 4, false), (d_bud, c_adap, g4, 5, false),
        (d_bud, c_pers, g1, 2, false), (d_bud, c_pers, g2, 2, false), (d_bud, c_pers, g3, 3, false), (d_bud, c_pers, g4, 4, false),
        (d_bud, c_resp, g1, 3, false), (d_bud, c_resp, g2, 4, true),  (d_bud, c_resp, g3, 5, true),  (d_bud, c_resp, g4, 7, true),
        (d_bud, c_crea, g1, 2, false), (d_bud, c_crea, g2, 2, false), (d_bud, c_crea, g3, 3, false), (d_bud, c_crea, g4, 3, false),
        (d_bud, c_ei,   g1, 3, false), (d_bud, c_ei,   g2, 4, false), (d_bud, c_ei,   g3, 4, false), (d_bud, c_ei,   g4, 5, false),
        (d_bud, c_team, g1, 3, false), (d_bud, c_team, g2, 3, false), (d_bud, c_team, g3, 4, false), (d_bud, c_team, g4, 6, false),
        (d_bud, c_deta, g1, 3, true),  (d_bud, c_deta, g2, 4, true),  (d_bud, c_deta, g3, 5, true),  (d_bud, c_deta, g4, 7, true),
        (d_bud, c_logi, g1, 3, true),  (d_bud, c_logi, g2, 4, true),  (d_bud, c_logi, g3, 5, true),  (d_bud, c_logi, g4, 6, true),
        (d_bud, c_anal, g1, 3, true),  (d_bud, c_anal, g2, 4, true),  (d_bud, c_anal, g3, 5, true),  (d_bud, c_anal, g4, 7, true),
        (d_bud, c_self, g1, 2, false), (d_bud, c_self, g2, 3, false), (d_bud, c_self, g3, 4, true),  (d_bud, c_self, g4, 6, true),
        (d_bud, c_deci, g1, 2, false), (d_bud, c_deci, g2, 3, false), (d_bud, c_deci, g3, 4, false), (d_bud, c_deci, g4, 6, true),
        (d_bud, c_lead, g2, 2, false), (d_bud, c_lead, g3, 3, false), (d_bud, c_lead, g4, 5, true);

    -- АХД
    INSERT INTO dept_competency_requirements (department_id, competency_id, grade_id, required_min, is_key) VALUES
        (d_ahd, c_comm, g1, 3, true),  (d_ahd, c_comm, g2, 3, true),  (d_ahd, c_comm, g3, 4, true),  (d_ahd, c_comm, g4, 5, true),
        (d_ahd, c_adap, g1, 2, false), (d_ahd, c_adap, g2, 3, false), (d_ahd, c_adap, g3, 4, false), (d_ahd, c_adap, g4, 5, false),
        (d_ahd, c_pers, g1, 2, false), (d_ahd, c_pers, g2, 3, false), (d_ahd, c_pers, g3, 4, false), (d_ahd, c_pers, g4, 5, false),
        (d_ahd, c_resp, g1, 3, false), (d_ahd, c_resp, g2, 4, false), (d_ahd, c_resp, g3, 5, true),  (d_ahd, c_resp, g4, 6, true),
        (d_ahd, c_crea, g1, 2, false), (d_ahd, c_crea, g2, 2, false), (d_ahd, c_crea, g3, 3, false), (d_ahd, c_crea, g4, 3, false),
        (d_ahd, c_ei,   g1, 3, false), (d_ahd, c_ei,   g2, 4, false), (d_ahd, c_ei,   g3, 4, true),  (d_ahd, c_ei,   g4, 5, true),
        (d_ahd, c_team, g1, 3, false), (d_ahd, c_team, g2, 3, false), (d_ahd, c_team, g3, 4, true),  (d_ahd, c_team, g4, 6, true),
        (d_ahd, c_deta, g1, 3, true),  (d_ahd, c_deta, g2, 4, true),  (d_ahd, c_deta, g3, 5, true),  (d_ahd, c_deta, g4, 7, true),
        (d_ahd, c_logi, g1, 3, false), (d_ahd, c_logi, g2, 4, false), (d_ahd, c_logi, g3, 5, false), (d_ahd, c_logi, g4, 6, false),
        (d_ahd, c_anal, g1, 2, false), (d_ahd, c_anal, g2, 3, false), (d_ahd, c_anal, g3, 4, false), (d_ahd, c_anal, g4, 5, false),
        (d_ahd, c_self, g1, 2, true),  (d_ahd, c_self, g2, 3, true),  (d_ahd, c_self, g3, 4, true),  (d_ahd, c_self, g4, 6, true),
        (d_ahd, c_deci, g1, 2, false), (d_ahd, c_deci, g2, 3, false), (d_ahd, c_deci, g3, 4, false), (d_ahd, c_deci, g4, 6, true),
        (d_ahd, c_lead, g2, 2, false), (d_ahd, c_lead, g3, 3, false), (d_ahd, c_lead, g4, 5, true);

    -- ДЗЛ
    INSERT INTO dept_competency_requirements (department_id, competency_id, grade_id, required_min, is_key) VALUES
        (d_dzl, c_comm, g1, 2, false), (d_dzl, c_comm, g2, 3, false), (d_dzl, c_comm, g3, 4, true),  (d_dzl, c_comm, g4, 5, true),
        (d_dzl, c_adap, g1, 2, false), (d_dzl, c_adap, g2, 3, false), (d_dzl, c_adap, g3, 4, false), (d_dzl, c_adap, g4, 5, false),
        (d_dzl, c_pers, g1, 2, false), (d_dzl, c_pers, g2, 3, false), (d_dzl, c_pers, g3, 4, false), (d_dzl, c_pers, g4, 5, false),
        (d_dzl, c_resp, g1, 3, true),  (d_dzl, c_resp, g2, 4, true),  (d_dzl, c_resp, g3, 5, true),  (d_dzl, c_resp, g4, 7, true),
        (d_dzl, c_crea, g1, 2, false), (d_dzl, c_crea, g2, 2, false), (d_dzl, c_crea, g3, 3, false), (d_dzl, c_crea, g4, 4, false),
        (d_dzl, c_ei,   g1, 2, false), (d_dzl, c_ei,   g2, 3, false), (d_dzl, c_ei,   g3, 4, false), (d_dzl, c_ei,   g4, 5, false),
        (d_dzl, c_team, g1, 3, false), (d_dzl, c_team, g2, 4, false), (d_dzl, c_team, g3, 4, false), (d_dzl, c_team, g4, 5, false),
        (d_dzl, c_deta, g1, 3, true),  (d_dzl, c_deta, g2, 4, true),  (d_dzl, c_deta, g3, 5, true),  (d_dzl, c_deta, g4, 7, true),
        (d_dzl, c_logi, g1, 3, false), (d_dzl, c_logi, g2, 4, false), (d_dzl, c_logi, g3, 5, false), (d_dzl, c_logi, g4, 7, true),
        (d_dzl, c_anal, g1, 3, true),  (d_dzl, c_anal, g2, 4, true),  (d_dzl, c_anal, g3, 5, true),  (d_dzl, c_anal, g4, 7, true),
        (d_dzl, c_self, g1, 3, false), (d_dzl, c_self, g2, 4, false), (d_dzl, c_self, g3, 5, true),  (d_dzl, c_self, g4, 7, true),
        (d_dzl, c_deci, g1, 2, false), (d_dzl, c_deci, g2, 3, false), (d_dzl, c_deci, g3, 4, true),  (d_dzl, c_deci, g4, 6, true),
        (d_dzl, c_lead, g2, 2, false), (d_dzl, c_lead, g3, 3, false), (d_dzl, c_lead, g4, 5, true);

    -- ДИТ
    INSERT INTO dept_competency_requirements (department_id, competency_id, grade_id, required_min, is_key) VALUES
        (d_dit, c_comm, g1, 2, false), (d_dit, c_comm, g2, 3, false), (d_dit, c_comm, g3, 4, false), (d_dit, c_comm, g4, 5, false),
        (d_dit, c_adap, g1, 2, false), (d_dit, c_adap, g2, 3, false), (d_dit, c_adap, g3, 4, false), (d_dit, c_adap, g4, 5, false),
        (d_dit, c_pers, g1, 2, false), (d_dit, c_pers, g2, 3, false), (d_dit, c_pers, g3, 4, false), (d_dit, c_pers, g4, 5, false),
        (d_dit, c_resp, g1, 3, false), (d_dit, c_resp, g2, 4, false), (d_dit, c_resp, g3, 5, true),  (d_dit, c_resp, g4, 7, true),
        (d_dit, c_crea, g1, 2, false), (d_dit, c_crea, g2, 3, false), (d_dit, c_crea, g3, 3, false), (d_dit, c_crea, g4, 4, false),
        (d_dit, c_ei,   g1, 2, false), (d_dit, c_ei,   g2, 3, false), (d_dit, c_ei,   g3, 3, false), (d_dit, c_ei,   g4, 4, false),
        (d_dit, c_team, g1, 3, false), (d_dit, c_team, g2, 4, false), (d_dit, c_team, g3, 5, true),  (d_dit, c_team, g4, 6, true),
        (d_dit, c_deta, g1, 3, true),  (d_dit, c_deta, g2, 4, true),  (d_dit, c_deta, g3, 5, true),  (d_dit, c_deta, g4, 7, true),
        (d_dit, c_logi, g1, 3, true),  (d_dit, c_logi, g2, 4, true),  (d_dit, c_logi, g3, 5, true),  (d_dit, c_logi, g4, 7, true),
        (d_dit, c_anal, g1, 2, false), (d_dit, c_anal, g2, 3, false), (d_dit, c_anal, g3, 5, true),  (d_dit, c_anal, g4, 6, true),
        (d_dit, c_self, g1, 3, true),  (d_dit, c_self, g2, 4, true),  (d_dit, c_self, g3, 5, true),  (d_dit, c_self, g4, 7, true),
        (d_dit, c_deci, g1, 3, false), (d_dit, c_deci, g2, 4, false), (d_dit, c_deci, g3, 5, false), (d_dit, c_deci, g4, 6, true),
        (d_dit, c_lead, g2, 2, false), (d_dit, c_lead, g3, 3, false), (d_dit, c_lead, g4, 5, true);

    -- ДЧР
    INSERT INTO dept_competency_requirements (department_id, competency_id, grade_id, required_min, is_key) VALUES
        (d_dcr, c_comm, g1, 3, false), (d_dcr, c_comm, g2, 4, false), (d_dcr, c_comm, g3, 6, true),  (d_dcr, c_comm, g4, 7, true),
        (d_dcr, c_adap, g1, 3, true),  (d_dcr, c_adap, g2, 4, true),  (d_dcr, c_adap, g3, 6, true),  (d_dcr, c_adap, g4, 7, true),
        (d_dcr, c_pers, g1, 3, false), (d_dcr, c_pers, g2, 4, false), (d_dcr, c_pers, g3, 5, false), (d_dcr, c_pers, g4, 6, false),
        (d_dcr, c_resp, g1, 3, false), (d_dcr, c_resp, g2, 4, false), (d_dcr, c_resp, g3, 6, true),  (d_dcr, c_resp, g4, 7, true),
        (d_dcr, c_crea, g1, 2, false), (d_dcr, c_crea, g2, 3, false), (d_dcr, c_crea, g3, 4, false), (d_dcr, c_crea, g4, 5, false),
        (d_dcr, c_ei,   g1, 3, true),  (d_dcr, c_ei,   g2, 4, true),  (d_dcr, c_ei,   g3, 6, true),  (d_dcr, c_ei,   g4, 7, true),
        (d_dcr, c_team, g1, 4, true),  (d_dcr, c_team, g2, 5, true),  (d_dcr, c_team, g3, 6, true),  (d_dcr, c_team, g4, 7, true),
        (d_dcr, c_deta, g1, 3, false), (d_dcr, c_deta, g2, 4, false), (d_dcr, c_deta, g3, 5, false), (d_dcr, c_deta, g4, 6, false),
        (d_dcr, c_logi, g1, 3, false), (d_dcr, c_logi, g2, 4, false), (d_dcr, c_logi, g3, 5, false), (d_dcr, c_logi, g4, 7, true),
        (d_dcr, c_anal, g1, 3, false), (d_dcr, c_anal, g2, 4, false), (d_dcr, c_anal, g3, 5, false), (d_dcr, c_anal, g4, 6, false),
        (d_dcr, c_self, g1, 3, false), (d_dcr, c_self, g2, 4, false), (d_dcr, c_self, g3, 5, true),  (d_dcr, c_self, g4, 7, true),
        (d_dcr, c_deci, g1, 2, false), (d_dcr, c_deci, g2, 3, false), (d_dcr, c_deci, g3, 4, false), (d_dcr, c_deci, g4, 6, false),
        (d_dcr, c_lead, g2, 3, false), (d_dcr, c_lead, g3, 4, false), (d_dcr, c_lead, g4, 7, true);
END $$;
