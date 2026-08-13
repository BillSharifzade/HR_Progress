-- Reverse 0016, as far as it can honestly be reversed.
--
-- The merge itself is NOT undone. Once `ДФП2`'s and `БИЮД`'s users, sections
-- and role grants have been repointed and the source rows deleted, nothing in
-- the schema records which department each row came from — re-splitting them
-- would be guesswork. If you need the pre-merge state back, restore from the
-- dump taken immediately before 0016 was applied (see backups/).
--
-- What this does reverse is the naming, so that `migrate down` followed by
-- `migrate up` is a clean round trip: the two departments go back to the
-- (incorrect) names migration 0005 seeded.
--
-- Note this deliberately does NOT re-soft-delete anything. 0016 revived `ДФП`
-- because a manual cleanup had soft-deleted the row holding the competency
-- matrix; putting that back would re-strand the matrix.

UPDATE departments
   SET name = 'Департамент Финансового Планирования'
 WHERE code = 'ДФП'
   AND name = 'Департамент Фармацевтической Промоции';

UPDATE departments
   SET name = 'Бюджетный Департамент'
 WHERE code = 'БЮД'
   AND name = 'Бухгалтерский и Юридический Департамент';
