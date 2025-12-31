import { UnitsResponse } from '../services/api/unitsAPI';
import { HexMarkers } from '../services/api/searchAPI';
import { GameTurn } from '../types/phaseTypes';
import { GameResponse } from '../types/gameTypes';

/**
 * Извлекает данные поиска (факторы и маркеры) из ViewModel
 * ViewModel уже содержит отфильтрованные данные для стороны игрока в search.search_hexes
 * @param response - Ответ API с данными ViewModel
 * @param playerSide - Сторона игрока ('german' или 'allied') - используется только для проверки, данные уже отфильтрованы
 * @returns Объект с картами факторов поиска и маркеров
 */
export function extractSearchDataFromModel(
  response: UnitsResponse,
  playerSide: 'german' | 'allied'
): { factorsMap: Map<string, number>; markersMap: Record<string, HexMarkers> } {
  const factorsMap = new Map<string, number>();
  const markersMap: Record<string, HexMarkers> = {};

  // ViewModel уже содержит отфильтрованные данные для стороны игрока в search.search_hexes
  const searchHexes = response.data?.search?.search_hexes;
  if (searchHexes) {
    Object.keys(searchHexes).forEach(hexId => {
      const hexSearchData = searchHexes[hexId];
      if (hexSearchData) {
        factorsMap.set(hexId, hexSearchData.factor || 0);
        if (hexSearchData.air_search > 0) {
          markersMap[hexId] = {
            flight_path_search: hexSearchData.air_search
          };
        }
      }
    });
  }

  return { factorsMap, markersMap };
}

/**
 * Создает объект GameTurn из данных GameModel
 * @param currentTurnData - Данные текущего хода из GameModel
 * @param currentGame - Текущая игра из store
 * @returns Объект GameTurn или null, если игра еще не начата
 */
export function createGameTurnFromModel(
  currentTurnData: { turn: number; phase: string } | undefined,
  currentGame: GameResponse | null
): GameTurn | null {
  if (!currentTurnData || currentTurnData.turn === 0) {
    return null;
  }

  return {
    id: `turn-${currentTurnData.turn}`,
    game_id: currentGame?.id || '',
    turn_number: currentTurnData.turn,
    current_phase: currentTurnData.phase as GameTurn['current_phase'],
    status: 'active',
    start_time: new Date().toISOString(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    visibility_level: currentGame?.visibility_level,
    is_fog: currentGame?.is_fog,
  };
}

/**
 * Обновляет все данные игры из GameModel
 * @param response - Ответ API с данными GameModel
 * @param currentGame - Текущая игра из store
 * @param playerSide - Сторона игрока ('german', 'allied' или 'unknown')
 * @param updateGame - Функция для обновления игры в store
 * @returns Объект с обновленными данными для установки в состояния компонента
 */
export function updateGameDataFromModel(
  response: UnitsResponse,
  currentGame: GameResponse | null,
  playerSide: string,
  updateGame: (gameId: string, updates: Partial<GameResponse>) => void
) {
  const result = {
    units: response.data?.units || [],
    taskForces: response.data?.task_forces || [],
    enemyContacts: response.data?.enemy_contacts || [],
    searchFactorHexes: new Map<string, number>(),
    hexMarkers: {} as Record<string, HexMarkers>,
    currentTurn: null as GameTurn | null,
  };

  // Извлекаем данные поиска
  if (playerSide === 'german' || playerSide === 'allied') {
    const { factorsMap, markersMap } = extractSearchDataFromModel(
      response,
      playerSide as 'german' | 'allied'
    );
    result.searchFactorHexes = factorsMap;
    result.hexMarkers = markersMap;
  }

  // ВАЖНО: Сначала обновляем currentGame в store, чтобы при создании GameTurn использовались актуальные значения
  // Собираем обновления, включая только определенные значения (не undefined)
  const gameUpdates: Partial<GameResponse> = {};
  
  // Создаем обновленный объект currentGame для использования при создании GameTurn
  let updatedCurrentGame: GameResponse | null = currentGame;
  
  if (currentGame?.id && response.data?.current_turn) {
    gameUpdates.current_turn = response.data.current_turn.turn;
    gameUpdates.current_phase = response.data.current_turn.phase;
    
    // Включаем visibility_level, is_fog, weather_track только если они определены
    if (response.data.visibility_level !== undefined) {
      gameUpdates.visibility_level = response.data.visibility_level;
    }
    if (response.data.is_fog !== undefined) {
      gameUpdates.is_fog = response.data.is_fog;
    }
    if (response.data.weather_track !== undefined) {
      gameUpdates.weather_track = response.data.weather_track;
    }
    
    // Обновляем currentGame в store
    updateGame(currentGame.id, gameUpdates);
    
    // Создаем обновленный объект для использования при создании GameTurn
    updatedCurrentGame = currentGame ? { ...currentGame, ...gameUpdates } : currentGame;
  } else if (currentGame?.id && !response.data?.current_turn) {
    // Если current_turn отсутствует, обновляем на setup
    gameUpdates.current_turn = 0;
    gameUpdates.current_phase = 'setup';
    
    // Включаем visibility_level, is_fog, weather_track только если они определены
    if (response.data.visibility_level !== undefined) {
      gameUpdates.visibility_level = response.data.visibility_level;
    }
    if (response.data.is_fog !== undefined) {
      gameUpdates.is_fog = response.data.is_fog;
    }
    if (response.data.weather_track !== undefined) {
      gameUpdates.weather_track = response.data.weather_track;
    }
    
    // Обновляем currentGame в store
    updateGame(currentGame.id, gameUpdates);
    
    // Создаем обновленный объект для использования при создании GameTurn
    updatedCurrentGame = currentGame ? { ...currentGame, ...gameUpdates } : currentGame;
  }

  // Создаем GameTurn ПОСЛЕ обновления currentGame, используя обновленные значения
  result.currentTurn = createGameTurnFromModel(
    response.data?.current_turn,
    updatedCurrentGame
  );

  return result;
}

