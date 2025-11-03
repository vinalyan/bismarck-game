-- Перенос данных из flight_path_search_markers в hex_markers
INSERT INTO hex_markers (game_id, player_id, hex_id, marker_type, created_at, updated_at)
SELECT game_id, player_id, hex_id, 'flight_path_search', created_at, updated_at
FROM flight_path_search_markers
ON CONFLICT DO NOTHING;

