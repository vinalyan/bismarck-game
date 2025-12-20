import { UnitsResponse } from '../services/api/unitsAPI';
import { HexMarkers } from '../services/api/searchAPI';
import { GameTurn } from '../types/phaseTypes';
import { GameResponse } from '../types/gameTypes';

/**
 * Извлекает данные поиска (факторы и маркеры) из GameModel
 * @param response - Ответ API с данными GameModel
 * @param playerSide - Сторона игрока ('german' или 'allied')
 * @returns Объект с картами факторов поиска и маркеров
 */
export function extractSearchDataFromModel(
  response: UnitsResponse,
  playerSide: 'german' | 'allied'
): { factorsMap: Map<string, number>; markersMap: Record<string, HexMarkers> } {
  const factorsMap = new Map<string, number>();
  const markersMap: Record<string, HexMarkers> = {};

  const searchData = response.data?.search?.[playerSide];
  if (searchData) {
    Object.keys(searchData).forEach(hexId => {
      const hexSearchData = searchData[hexId];
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

  // Создаем GameTurn
  result.currentTurn = createGameTurnFromModel(
    response.data?.current_turn,
    currentGame
  );

  // Обновляем currentGame в store
  if (currentGame?.id && response.data?.current_turn) {
    updateGame(currentGame.id, {
      current_turn: response.data.current_turn.turn,
      current_phase: response.data.current_turn.phase,
    });
  } else if (currentGame?.id && !response.data?.current_turn) {
    // Если current_turn отсутствует, обновляем на setup
    updateGame(currentGame.id, {
      current_turn: 0,
      current_phase: 'setup',
    });
  }

  return result;
}

