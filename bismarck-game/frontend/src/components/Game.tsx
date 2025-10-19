// Основной компонент игры "Погоня за Бисмарком"

import React, { useState, useEffect } from 'react';
import { useGameStore } from '../stores/gameStore';
import { ViewType, GamePhase, PlayerSide, NotificationType } from '../types/gameTypes';
import { HexCoordinate, coordinateToOffset, offsetToCoordinate } from '../types/mapTypes';
import { shipUtils, LOCAL_SHIPS_DATA, localShipsUtils, ShipData } from '../data/localShips';
import { movementUtils, MovementHex } from '../utils/movementUtils';
import { activeHexesUtils, ActiveHex, useActiveHexes } from '../utils/activeHexesUtils';
import { MAP_CONSTANTS } from '../utils/hexUtils';
import { unitsAPI, GameUnit, UpdatePositionRequest } from '../services/api/unitsAPI';
import { phaseAPI, GameTurn } from '../services/api/phaseAPI';
import { GamePhase as PhaseType } from '../types/phaseTypes';
import HexMap from './HexMap';
import PhasePanel from './PhasePanel';
import './Game.css';

const Game: React.FC = () => {
  const {
    user,
    currentGame,
    authToken,
    logout,
    setCurrentView,
    addNotification,
    setLoading,
  } = useGameStore();

  const [selectedHex, setSelectedHex] = useState<HexCoordinate | null>(null);
  const [selectedUnit, setSelectedUnit] = useState<string | null>(null);
  const [selectedUnitData, setSelectedUnitData] = useState<any>(null);
  const [availableMovementHexes, setAvailableMovementHexes] = useState<MovementHex[]>([]);
  const [shipsData, setShipsData] = useState<ShipData[]>([]);
  const [currentTurn, setCurrentTurn] = useState<GameTurn | null>(null);
  const [loadingShips, setLoadingShips] = useState(false);
  const [gameUnits, setGameUnits] = useState<GameUnit[]>([]);
  const [loadingUnits, setLoadingUnits] = useState(false);
  
  // Состояние для позиций юнитов
  const [unitPositions, setUnitPositions] = useState<Map<string, HexCoordinate>>(new Map());

  // Хук для управления активными гексами
  const {
    activeHexes,
    enabledTypes,
    addActiveHexes,
    removeActiveHexesByType,
    clearActiveHexes,
    toggleType
  } = useActiveHexes();

  // Загружаем данные кораблей и юнитов при монтировании компонента
  useEffect(() => {
    // Используем локальные данные для конфигурации кораблей
    setShipsData(LOCAL_SHIPS_DATA);
    setLoadingShips(false);

    // Загружаем юниты игры из API
    const loadGameUnits = async () => {
      if (!currentGame?.id || !authToken) {
        return;
      }

      try {
        setLoadingUnits(true);
        const response = await unitsAPI.getGameUnits(currentGame.id, authToken);
        
        if (response.success && response.data && response.data.units) {
          setGameUnits(response.data.units);
          
          // Создаем карту позиций юнитов
          const positions = new Map<string, HexCoordinate>();
          response.data.units.forEach(unit => {
            if (unit.position && unit.position.trim() !== '') {
              // Парсим позицию (например, "J30" -> {letter: "J", number: 30})
              const match = unit.position.match(/^([A-Z]+)(\d+)$/);
              if (match) {
                const letter = match[1];
                const number = parseInt(match[2]);
                
                // Преобразуем букву в row
                let row: number;
                if (letter.length === 1) {
                  // A, B, C, ..., Z (0-25)
                  row = letter.charCodeAt(0) - 65;
                } else if (letter.length === 2 && letter.startsWith('A')) {
                  // AA, AB, AC, ..., AH (26-33)
                  row = 26 + (letter.charCodeAt(1) - 65);
                } else {
                  console.warn(`Invalid letter format: ${letter}`);
                  return;
                }
                
                // Создаем координату
                const coordinate: HexCoordinate = {
                  letter: letter,
                  number: number,
                  col: number - 1,
                  row: row
                };
                positions.set(unit.id, coordinate);
              }
            }
          });
          setUnitPositions(positions);
        } else {
          console.error('Failed to load game units:', response.error);
          addNotification({
            type: NotificationType.Error,
            title: 'Ошибка загрузки юнитов',
            message: response.error || 'Не удалось загрузить юниты игры',
            read: false
          });
        }
      } catch (error) {
        console.error('Error loading game units:', error);
        addNotification({
          type: NotificationType.Error,
          title: 'Ошибка загрузки юнитов',
          message: 'Произошла ошибка при загрузке юнитов игры',
          read: false
        });
      } finally {
        setLoadingUnits(false);
      }
    };

    loadGameUnits();
  }, [currentGame?.id, authToken, addNotification]);

  // Обработчик клика по гексу для движения
  const handleHexClick = (coordinate: HexCoordinate) => {
    // Проверяем, есть ли выбранный юнит
    if (!selectedUnit || !selectedUnitData) {
      return;
    }

    // Проверяем, является ли гекс активным для движения
    const isMovementHex = activeHexes.some(hex => 
      hex.type === 'movement' &&
      hex.coordinate.col === coordinate.col && 
      hex.coordinate.row === coordinate.row
    );

    if (isMovementHex) {
      // Выполняем движение
      handleMovement(coordinate);
    }
  };

  // Обработчик движения
  const handleMovement = async (targetCoordinate: HexCoordinate) => {
    if (!selectedUnitData || !currentGame?.id || !authToken) {
      return;
    }

    // Создаем объект с данными корабля из API для расчета движения
    const shipDataFromAPI: ShipData = {
      id: selectedUnitData.id,
      name: selectedUnitData.name,
      type: selectedUnitData.type,
      side: selectedUnitData.side,
      maxFuel: selectedUnitData.max_fuel,
      baseEvasion: selectedUnitData.base_evasion,
      radarLevel: selectedUnitData.radar_level || 0,
      hullBoxes: selectedUnitData.hull_boxes || 0,
      basePrimaryArmamentBow: selectedUnitData.base_primary_armament_bow || 0,
      basePrimaryArmamentStern: selectedUnitData.base_primary_armament_stern || 0,
      baseSecondaryArmament: selectedUnitData.base_secondary_armament || 0,
      maxTorpedos: selectedUnitData.max_torpedoes || 0,
      speedType: selectedUnitData.speed_rating,
      setupHex: selectedUnitData.setup_hex
    };

    // Создаем информацию о предыдущем ходе (пока заглушка, нужно получать с сервера)
    const previousTurnInfo = {
      movedHexes: selectedUnitData.previous_turn_moved_hexes || 0,
      turnNumber: selectedUnitData.last_move_turn || 0
    };

    // Отладочная информация
    console.log('🔍 Отладка движения Bismarck:');
    console.log('  - previous_turn_moved_hexes:', selectedUnitData.previous_turn_moved_hexes);
    console.log('  - last_move_turn:', selectedUnitData.last_move_turn);
    console.log('  - moved_hexes:', selectedUnitData.moved_hexes);
    console.log('  - previousTurnInfo:', previousTurnInfo);

    // Рассчитываем стоимость движения согласно правилам игры
    const movementCost = movementUtils.canReachHex(
      shipDataFromAPI,
      selectedUnitData.position,
      targetCoordinate,
      selectedUnitData.currentFuel,
      previousTurnInfo
    );

    if (!movementCost.canReach) {
      console.log('Невозможно добраться до этого гекса:', movementCost.reason);
      addNotification({
        type: NotificationType.Error,
        title: 'Движение невозможно',
        message: movementCost.reason || 'Неизвестная причина',
        read: false
      });
      return;
    }

    const newFuel = selectedUnitData.currentFuel - movementCost.fuelCost;
    const positionString = `${targetCoordinate.letter}${targetCoordinate.number}`;

    try {
      // Сохраняем позицию на сервере
      const updateRequest: UpdatePositionRequest = {
        position: positionString,
        fuel: newFuel,
        hexesMoved: movementCost.distance // Отправляем количество гексов, на которое переместился юнит
      };

      const response = await unitsAPI.updateUnitPosition(
        currentGame.id,
        selectedUnit!,
        updateRequest,
        authToken
      );

      if (response.success) {
        // Обновляем позицию юнита локально
        const updatedUnitData = {
          ...selectedUnitData,
          position: targetCoordinate,
          currentFuel: newFuel
        };

        setSelectedUnitData(updatedUnitData);

        // Обновляем позицию юнита в состоянии
        setUnitPositions(prev => {
          const newPositions = new Map(prev);
          newPositions.set(selectedUnit!, targetCoordinate);
          return newPositions;
        });

        // Обновляем данные юнита в gameUnits
        setGameUnits(prev => prev.map(unit => 
          unit.id === selectedUnit 
            ? { ...unit, position: positionString, fuel: newFuel }
            : unit
        ));

        // Очищаем активные гексы
        clearActiveHexes();
        setAvailableMovementHexes([]);

        // Обновляем информацию о предыдущем ходе для следующего движения
        const updatedPreviousTurnInfo = {
          movedHexes: movementCost.distance, // Текущее движение становится предыдущим
          turnNumber: (previousTurnInfo?.turnNumber || 0) + 1
        };

        // Пересчитываем доступные гексы для движения с новой позиции
        const newAvailableHexes = movementUtils.getAvailableMovementHexes(
          selectedUnitData.shipData,
          targetCoordinate,
          newFuel,
          updatedPreviousTurnInfo
        );
        setAvailableMovementHexes(newAvailableHexes);

        // Добавляем новые активные гексы
        const newMovementActiveHexes = activeHexesUtils.getMovementActiveHexes(
          selectedUnitData.shipData,
          targetCoordinate,
          newFuel,
          updatedPreviousTurnInfo
        );
        addActiveHexes(newMovementActiveHexes);

        console.log(`Юнит ${selectedUnit} перемещен в ${targetCoordinate.letter}${targetCoordinate.number}`);
        console.log(`Потрачено топлива: ${movementCost.fuelCost}, осталось: ${newFuel}`);
        
        addNotification({
          type: NotificationType.Success,
          title: 'Движение выполнено',
          message: `Юнит перемещен в ${positionString}`,
          read: false
        });
      } else {
        console.error('Failed to update unit position:', response.error);
        addNotification({
          type: NotificationType.Error,
          title: 'Ошибка движения',
          message: response.error || 'Не удалось сохранить позицию юнита',
          read: false
        });
      }
    } catch (error) {
      console.error('Error updating unit position:', error);
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка движения',
        message: 'Произошла ошибка при сохранении позиции юнита',
        read: false
      });
    }
  };

  // Обработчик клика по юниту
  const handleUnitClick = async (unitId: string, unitData: any) => {
    setSelectedUnit(unitId);
    setSelectedUnitData(unitData);

    // Очищаем предыдущие активные гексы
    clearActiveHexes();

    // Получаем актуальную позицию юнита
    const currentPosition = unitPositions.get(unitId) || unitData.position;

    // Находим данные юнита в gameUnits
    const gameUnit = gameUnits.find(unit => unit.id === unitId);
    
    // Пытаемся найти данные корабля в локальных данных
    let shipData = shipsData.find(ship => 
      ship.type === (gameUnit?.type || unitData.type) && 
      ship.side === (gameUnit?.nationality || unitData.side)
    );

    // Если не нашли, используем локальные утилиты
    if (!shipData && (gameUnit?.type || unitData.type) && (gameUnit?.nationality || unitData.side)) {
      const ships = localShipsUtils.getShipsByType(gameUnit?.type || unitData.type);
      shipData = ships.find(ship => ship.side === (gameUnit?.nationality || unitData.side));
    }

    // Обновляем данные юнита с информацией о корабле
    if (shipData) {
      const updatedUnitData = {
        ...unitData,
        ...gameUnit, // Добавляем данные из API
        position: currentPosition, // Используем актуальную позицию
        shipData: shipData,
        maxFuel: shipData.maxFuel,
        currentFuel: gameUnit?.fuel || unitData.fuel || Math.floor(shipData.maxFuel * 0.85) // Используем реальное топливо из API
      };
      setSelectedUnitData(updatedUnitData);

      // Рассчитываем доступные гексы для движения
      if (currentPosition) {
        // Создаем информацию о предыдущем ходе
        const previousTurnInfo = {
          movedHexes: updatedUnitData.previous_turn_moved_hexes || 0,
          turnNumber: updatedUnitData.last_move_turn || 0
        };

        const availableHexes = movementUtils.getAvailableMovementHexes(
          shipData,
          currentPosition,
          updatedUnitData.currentFuel,
          previousTurnInfo
        );
        setAvailableMovementHexes(availableHexes);

        // Добавляем активные гексы для движения
        const movementActiveHexes = activeHexesUtils.getMovementActiveHexes(
          shipData,
          currentPosition,
          updatedUnitData.currentFuel,
          previousTurnInfo
        );
        addActiveHexes(movementActiveHexes);
      }
    } else {
      setAvailableMovementHexes([]);
    }
  };

  // Заглушка для карты (пока без реальной гексагональной карты)
  const generateHexGrid = () => {
    const hexes = [];
    const rows = 15;
    const cols = 20;
    
    for (let row = 0; row < rows; row++) {
      for (let col = 0; col < cols; col++) {
        const hexId = `${row}-${col}`;
        const isWater = Math.random() > 0.3; // 70% воды
        const hasUnit = Math.random() > 0.95; // 5% шанс наличия юнита
        
        hexes.push({
          id: hexId,
          row,
          col,
          isWater,
          hasUnit,
          unitType: hasUnit ? (Math.random() > 0.5 ? 'battleship' : 'destroyer') : null,
          side: hasUnit ? (Math.random() > 0.5 ? PlayerSide.German : PlayerSide.Allied) : null,
        });
      }
    }
    return hexes;
  };

  const [hexGrid] = useState(generateHexGrid());

  // Определяем сторону игрока
  const playerSide = currentGame?.player1_id === user?.id 
    ? currentGame?.player1_side 
    : currentGame?.player2_side;

  // Отладочная информация
  console.log('Debug Game Info:', {
    userId: user?.id,
    player1Id: currentGame?.player1_id,
    player2Id: currentGame?.player2_id,
    player1Side: currentGame?.player1_side,
    player2Side: currentGame?.player2_side,
    calculatedPlayerSide: playerSide,
    isPlayer1: currentGame?.player1_id === user?.id
  });

  const opponentSide = playerSide === PlayerSide.German 
    ? PlayerSide.Allied 
    : PlayerSide.German;

  // Получаем информацию о текущей фазе
  const getCurrentPhaseText = (phase: GamePhase): string => {
    switch (phase) {
      case GamePhase.Waiting:
        return 'Ожидание начала';
      case GamePhase.Setup:
        return 'Подготовка';
      case GamePhase.Movement:
        return 'Фаза движения';
      case GamePhase.Search:
        return 'Фаза поиска';
      case GamePhase.Combat:
        return 'Боевая фаза';
      case GamePhase.End:
        return 'Конец игры';
      default:
        return 'Неизвестная фаза';
    }
  };

  // Обработчик заправки всех кораблей
  const handleRefuelAllShips = async () => {
    if (!currentGame?.id || !authToken || gameUnits.length === 0) {
      return;
    }

    try {
      // Обновляем топливо для всех кораблей
      const updatedUnits = gameUnits.map(unit => {
        const newFuel = Math.min(unit.fuel + 4, unit.max_fuel || 18); // Не превышаем максимальное топливо
        return { ...unit, fuel: newFuel };
      });

      // Обновляем состояние
      setGameUnits(updatedUnits);

      // Обновляем выбранный юнит, если он есть
      if (selectedUnitData) {
        const updatedSelectedUnit = updatedUnits.find(unit => unit.id === selectedUnit);
        if (updatedSelectedUnit) {
          setSelectedUnitData({
            ...selectedUnitData,
            currentFuel: updatedSelectedUnit.fuel
          });
        }
      }

      // Показываем уведомление
      addNotification({
        type: NotificationType.Success,
        title: 'Заправка завершена',
        message: `Все корабли получили +4 топлива`,
        read: false
      });

      console.log('Все корабли заправлены на +4 топлива');
    } catch (error) {
      console.error('Error refueling ships:', error);
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка заправки',
        message: 'Произошла ошибка при заправке кораблей',
        read: false
      });
    }
  };

  // Возврат в лобби
  const handleBackToLobby = () => {
    setCurrentView(ViewType.Lobby);
  };

  // Выход из игры
  const handleLogout = () => {
    logout();
    setCurrentView(ViewType.Login);
  };

  if (!currentGame || !user) {
    return (
      <div className="game-container">
        <div className="game-error">
          <h2>Ошибка</h2>
          <p>Игра не найдена или пользователь не авторизован</p>
          <button onClick={handleBackToLobby} className="back-button">
            Вернуться в лобби
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="game-container">
      {/* Заголовок игры */}
      <div className="game-header">
        <div className="game-title">
          <h1>🎮 {currentGame.name}</h1>
          <div className="game-info">
            <span className="phase-info">
              Фаза: {getCurrentPhaseText(currentGame.current_phase)}
            </span>
            <span className="turn-info">
              Ход: {currentGame.current_turn}
            </span>
          </div>
        </div>
        
        <div className="game-controls">
          <div className="player-info">
            <span className="player-side">
              Ваша сторона: {playerSide === PlayerSide.German ? '🇩🇪 Немцы' : '🇬🇧 Союзники'}
            </span>
          </div>
          <button onClick={handleBackToLobby} className="back-button">
            ← Лобби
          </button>
          <button onClick={handleLogout} className="logout-button">
            Выйти
          </button>
        </div>
      </div>

      {/* Основной контент игры */}
      <div className="game-content">
        {/* Левая панель - информация об игре */}
        <div className="game-sidebar">
          {/* Информация о юнитах игрока */}
          <div className="units-info">
            <h3>Ваши юниты</h3>
            {loadingUnits ? (
              <div className="loading">Загрузка юнитов...</div>
            ) : gameUnits.filter(unit => unit.position && unit.position.trim() !== '').length > 0 ? (
            <div className="unit-list">
                {gameUnits
                  .filter(unit => unit.position && unit.position.trim() !== '')
                  .map((unit) => (
                    <div key={unit.id} className="unit-item">
                      <div className="unit-header">
                        <span className="unit-name">{unit.name}</span>
                        <span className="unit-type">{unit.type}</span>
                      </div>
                      <div className="unit-status">
                        <span>Позиция: {unit.position}</span>
                        <span>Топливо: {unit.fuel || 0}/{unit.max_fuel || 0}</span>
                        <span>Скорость: {
                          unit.speed_rating === 'F' ? 'Быстрый' :
                          unit.speed_rating === 'M' ? 'Средний' :
                          unit.speed_rating === 'S' ? 'Медленный' :
                          unit.speed_rating === 'VS' ? 'Очень медленный' :
                          unit.speed_rating || 'Неизвестно'
                        }</span>
              </div>
              </div>
                  ))}
              </div>
            ) : (
              <div className="no-units">Нет юнитов на карте</div>
            )}
          </div>

          {/* Панель управления фазами */}
          <PhasePanel gameId={currentGame.id} />

          {/* Выбранный гекс/юнит */}
          {selectedHex && (
            <div className="selected-info">
              <h3>Выбранная позиция</h3>
              <div className="hex-info">
                <span>Координаты: {selectedHex.letter}{selectedHex.number}</span>
                {selectedUnit && (
                  <div className="unit-details">
                    <span>Юнит: {selectedUnit}</span>
                    <span>Сторона: {PlayerSide.German}</span>
                    <span>Состояние: Активен</span>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Центральная область - карта */}
        <div className="game-map">
          
          <HexMap
            width={MAP_CONSTANTS.HEX_GRID_WIDTH}
            height={MAP_CONSTANTS.HEX_GRID_HEIGHT}
            playerSide={playerSide === PlayerSide.German ? 'german' : 'allied'}
            onHexClick={handleHexClick}
            onHexHover={(coordinate: HexCoordinate) => {
              // Можно добавить логику подсветки при наведении
            }}
            onUnitClick={handleUnitClick}
            selectedHex={selectedHex}
            availableMovementHexes={availableMovementHexes}
            activeHexes={activeHexes}
            unitPositions={unitPositions}
            gameUnits={gameUnits}
          />
        </div>

        {/* Правая панель - действия */}
        <div className="game-actions">
          
          {/* Информация о выбранном юните */}
          {selectedUnitData && (
            <div className="unit-info">
              <h3>Выбранный юнит</h3>
              <div className="unit-details">
                <div className="detail-item">
                  <span>Название:</span>
                  <span className="detail-value">{selectedUnitData.name}</span>
                </div>
                <div className="detail-item">
                  <span>Тип:</span>
                  <span className="detail-value">{selectedUnitData.type}</span>
                </div>
                <div className="detail-item">
                  <span>Позиция:</span>
                  <span className="detail-value">{selectedUnitData.position ? `${selectedUnitData.position.letter}${selectedUnitData.position.number}` : 'Неизвестно'}</span>
                </div>
                <div className="detail-item">
                  <span>Топливо:</span>
                  <span className="detail-value">
                    {selectedUnitData?.currentFuel || 85}/{selectedUnitData?.maxFuel || 100}
                  </span>
                  </div>
                {selectedUnitData && (
                  <>
                    <div className="detail-item">
                      <span>Скорость:</span>
                      <span className="detail-value">
                        {selectedUnitData.speed_rating === 'F' ? 'Быстрый' :
                         selectedUnitData.speed_rating === 'M' ? 'Средний' :
                         selectedUnitData.speed_rating === 'S' ? 'Медленный' :
                         selectedUnitData.speed_rating === 'VS' ? 'Очень медленный' :
                         selectedUnitData.speed_rating}
                      </span>
                  </div>
                    <div className="detail-item">
                      <span>Уклонение:</span>
                      <span className="detail-value">{selectedUnitData.base_evasion}</span>
                  </div>
                    <div className="detail-item">
                      <span>Радар:</span>
                      <span className="detail-value">
                        {selectedUnitData.radar_level === 0 ? 'Нет радара' :
                         selectedUnitData.radar_level === 1 ? 'RADAR I' :
                         selectedUnitData.radar_level === 2 ? 'RADAR II' :
                         `RADAR ${selectedUnitData.radar_level}`}
                    </span>
                  </div>
                  </>
                )}
              </div>

              
              {/* Доступные действия */}
              <div className="unit-actions">
                <h4>Доступные действия:</h4>
                <div className="action-buttons">
                  <button 
                    className="action-button"
                    onClick={() => {
                      // TODO: Реализовать оперативные соединения
                      console.log('Оперативные соединения для юнита:', selectedUnit);
                    }}
                  >
                    ⚓ Оперативные соединения
                  </button>
                  <button 
                    className="action-button"
                    onClick={() => {
                      // TODO: Реализовать заправку
                      console.log('Заправка для юнита:', selectedUnit);
                    }}
                  >
                    ⛽ Заправка
                  </button>
                  <button 
                    className="action-button"
                    onClick={() => {
                      // TODO: Реализовать патруль
                      console.log('Патруль для юнита:', selectedUnit);
                    }}
                  >
                    🛡️ Заявить патруль
                  </button>
                  <button 
                    className="action-button"
                    onClick={() => {
                      // TODO: Реализовать ремонт
                      console.log('Ремонт для юнита:', selectedUnit);
                    }}
                  >
                    🛠️ Попытка ремонта
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Информация о выбранном гексе */}
          {selectedHex && !selectedUnitData && (
            <div className="hex-info">
              <h3>Выбранный гекс</h3>
              <div className="hex-details">
                <div className="hex-item">
                  <span>Координата:</span>
                  <span className="hex-value">{selectedHex.letter}{selectedHex.number}</span>
                </div>
              </div>
            </div>
          )}
          
          <div className="action-panel">
            <h3>Действия</h3>
            <div className="action-buttons">
              <button 
                className="action-button"
                onClick={handleRefuelAllShips}
                disabled={!currentGame || gameUnits.length === 0}
              >
                Заправить (+4 топлива всем кораблям)
              </button>
              <button className="action-button">
                Завершить ход
              </button>
            </div>
          </div>

          {/* Информация о погоде */}
          <div className="weather-info">
            <h3>Погода</h3>
            <div className="weather-item">
              <span>Видимость:</span>
              <span className="weather-value">Хорошая</span>
            </div>
            <div className="weather-item">
              <span>Ветер:</span>
              <span className="weather-value">Умеренный</span>
            </div>
            <div className="weather-item">
              <span>Волнение:</span>
              <span className="weather-value">Слабое</span>
            </div>
          </div>
        </div>
      </div>

    </div>
  );
};

export default Game;
