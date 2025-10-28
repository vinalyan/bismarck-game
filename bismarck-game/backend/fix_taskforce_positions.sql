-- Исправление позиций кораблей в Task Force
-- Корабли в составе TF не должны иметь собственной позиции

-- 1. Сначала посмотрим на текущее состояние
SELECT 'BEFORE CLEANUP - Naval units in Task Force:' as status;
SELECT u.id, u.name, u.position, tf.name as task_force_name, tf.id as tf_id 
FROM naval_units u 
JOIN task_forces tf ON u.task_force_id = tf.id 
WHERE u.task_force_id IS NOT NULL
ORDER BY tf.name, u.name;

SELECT 'BEFORE CLEANUP - Task Forces:' as status;
SELECT id, name, position, units FROM task_forces ORDER BY name;

-- 2. Обнуляем позиции всех кораблей в Task Force
UPDATE naval_units 
SET position = '' 
WHERE task_force_id IS NOT NULL;

-- 3. Проверяем результат
SELECT 'AFTER CLEANUP - Naval units with cleared positions:' as status;
SELECT u.id, u.name, u.position, tf.name as task_force_name 
FROM naval_units u 
JOIN task_forces tf ON u.task_force_id = tf.id 
WHERE u.task_force_id IS NOT NULL
ORDER BY tf.name, u.name;

-- 4. Показываем Task Forces и их позиции
SELECT 'Task Forces positions:' as status;
SELECT id, name, position, units FROM task_forces ORDER BY name;
