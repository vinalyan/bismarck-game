-- Удаление старой таблицы flight_path_search_markers
-- После миграции данных в hex_markers старая таблица больше не нужна
DROP TABLE IF EXISTS flight_path_search_markers;

