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
import { refuelAPI } from '../services/api/refuelAPI';
import { GameTurnResponse } from '../types/phaseTypes';
import { GamePhase as PhaseType } from '../types/phaseTypes';
import HexMap from './HexMap';
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
  const [currentTurn, setCurrentTurn] = useState<GameTurn | GameTurnResponse | null>(null);
  const [phaseTimer, setPhaseTimer] = useState<number | null>(null);

  // Helper функция для получения данных хода
  const getTurnData = (turn: GameTurn | GameTurnResponse | null): GameTurn | null => {
    if (!turn) return null;
    if ('data' in turn && turn.data) {
      return turn.data;
    }
    if ('turn_number' in turn) {
      return turn as GameTurn;
    }
    return null;
  };

  // Helper функция для отображения названий фаз
  const getPhaseDisplayName = (phase: string): string => {
    const phaseNames: { [key: string]: string } = {
      'setup': 'Подготовка',
      'visibility': 'Видимость',
      'pursuit': 'Преследование',
      'movement': 'Движение',
      'search': 'Поиск',
      'air_attack': 'Воздушная атака',
      'naval_combat': 'Морской бой',
      'chance': 'Случайное событие',
      'admin': 'Администрирование'
    };
    return phaseNames[phase] || phase;
  };
  const [loadingShips, setLoadingShips] = useState(false);
  const [gameUnits, setGameUnits] = useState<GameUnit[]>([]);
  const [loadingUnits, setLoadingUnits] = useState(false);

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

  // Загружаем информацию о текущем ходе при монтировании компонента
  useEffect(() => {
    const loadCurrentTurn = async () => {
      if (!currentGame?.id) {
        return;
      }

      try {
        const turn = await phaseAPI.getCurrentPhase(currentGame.id);
        setCurrentTurn(turn);
      } catch (error) {
        console.error('Error loading current turn:', error);
        // Если нет активного хода, устанавливаем null
        setCurrentTurn(null);
      }
    };

    loadCurrentTurn();
  }, [currentGame?.id]);

  // Автоматическое обновление информации о текущей фазе каждые 2 секунды
  useEffect(() => {
    if (!currentGame?.id) {
      return;
    }

    const interval = setInterval(async () => {
      try {
        const turn = await phaseAPI.getCurrentPhase(currentGame.id);
        const previousTurn = currentTurn;
        
        setCurrentTurn(turn);
        
        // Проверяем, изменилась ли фаза
        if (previousTurn && turn) {
          const previousTurnData = getTurnData(previousTurn);
          const currentTurnData = getTurnData(turn);
          
          if (previousTurnData && currentTurnData) {
            // Если изменилась фаза или ход
            if (previousTurnData.current_phase !== currentTurnData.current_phase ||
                previousTurnData.turn_number !== currentTurnData.turn_number) {
              
              // Показываем уведомление о смене фазы
              if (previousTurnData.current_phase !== currentTurnData.current_phase) {
                addNotification({
                  type: NotificationType.Info,
                  title: 'Смена фазы',
                  message: `Переход к фазе: ${getPhaseDisplayName(currentTurnData.current_phase)}`,
                  read: false
                });
              }
              
              // Показываем уведомление о новом ходе
              if (previousTurnData.turn_number !== currentTurnData.turn_number) {
                addNotification({
                  type: NotificationType.Success,
                  title: 'Новый ход',
                  message: `Начат ход ${currentTurnData.turn_number}`,
                  read: false
                });
              }
            }
          }
        }
      } catch (error) {
        console.error('Error updating current turn:', error);
      }
    }, 2000); // Обновляем каждые 2 секунды

    return () => clearInterval(interval);
  }, [currentGame?.id, currentTurn, addNotification]);

  // Таймер обратного отсчета для автоматических фаз
  useEffect(() => {
    const turnData = getTurnData(currentTurn);
    if (!turnData || !turnData.current_phase) {
      setPhaseTimer(null);
      return;
    }

    // Фазы с автоматическим переходом
    const autoTransitionPhases = ['visibility', 'pursuit', 'search', 'air_attack', 'naval_combat', 'chance', 'admin'];
    
    if (autoTransitionPhases.includes(turnData.current_phase)) {
      setPhaseTimer(1); // 1 секунда до автоматического перехода (соответствует бэкенду)
      
      const timer = setInterval(() => {
        setPhaseTimer(prev => {
          if (prev === null || prev <= 1) {
            return null;
          }
          return prev - 1;
        });
      }, 1000);

      return () => clearInterval(timer);
    } else {
      setPhaseTimer(null);
    }
  }, [currentTurn]);

  // Обработчик обновления хода
  useEffect(() => {
    const handleTurnUpdate = (event: CustomEvent) => {
      const updatedTurn = event.detail;
      setCurrentTurn(updatedTurn);
    };

    window.addEventListener('turnUpdated', handleTurnUpdate as EventListener);
    
    return () => {
      window.removeEventListener('turnUpdated', handleTurnUpdate as EventListener);
    };
  }, []);

  // Обработчик обновления игры
  useEffect(() => {
    const handleGameUpdate = async () => {
      // Перезагружаем информацию об игре из store
      // Это обновит currentGame.current_phase
      if (currentGame?.id) {
        // Можно добавить API вызов для обновления информации об игре
      }
    };

    window.addEventListener('gameUpdated', handleGameUpdate as EventListener);
    
    return () => {
      window.removeEventListener('gameUpdated', handleGameUpdate as EventListener);
    };
  }, [currentGame?.id]);


  // Обработчик движения
  const handleMovement = async (targetCoordinate: HexCoordinate) => {
    console.log('handleMovement called:', targetCoordinate);
    console.log('selectedUnitData:', selectedUnitData);
    console.log('currentGame:', currentGame);
    console.log('authToken:', authToken);
    
    if (!selectedUnitData || !currentGame?.id || !authToken) {
      console.log('Missing required data for movement');
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


    // Рассчитываем стоимость движения согласно правилам игры
    const movementCost = movementUtils.canReachHex(
      shipDataFromAPI,
      selectedUnitData.position,
      targetCoordinate,
      selectedUnitData.currentFuel,
      previousTurnInfo
    );

    if (!movementCost.canReach) {
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

      console.log('Sending movement request:', {
        gameId: currentGame.id,
        unitId: selectedUnit!,
        updateRequest,
        authToken: authToken ? 'present' : 'missing'
      });
      
      const response = await unitsAPI.updateUnitPosition(
        currentGame.id,
        selectedUnit!,
        updateRequest,
        authToken
      );
      
      console.log('Movement response:', response);

      if (response.success) {
        // Обновляем позицию юнита локально
        const movementRestriction = (selectedUnitData.speed_rating === 'S') ? 2 : 
                                   (selectedUnitData.speed_rating === 'VS') ? 4 : 0;
        
        const updatedUnitData = {
          ...selectedUnitData,
          position: targetCoordinate,
          currentFuel: newFuel,
          movement_used: movementCost.distance, // Устанавливаем использованное движение
          last_move_turn: getTurnData(currentTurn)?.turn_number || 0, // Устанавливаем ход движения
          // Для S и VS типов устанавливаем ограничения движения
          no_movement_turns_left: movementRestriction
        };

        console.log('Movement completed, setting restrictions:', {
          unitName: selectedUnitData.name,
          speedRating: selectedUnitData.speed_rating,
          movementRestriction: movementRestriction
        });

        // Добавляем уведомление о блокировке движения
        if (movementRestriction > 0) {
          addNotification({
            type: NotificationType.Info,
            title: 'Движение заблокировано',
            message: `${selectedUnitData.name} заблокирован на ${movementRestriction} ход${movementRestriction > 1 ? 'а' : ''} из-за ограничений скорости`,
            read: false
          });
        }

        setSelectedUnitData(updatedUnitData);

        // Обновляем данные юнита в gameUnits
        setGameUnits(prev => prev.map(unit => 
          unit.id === selectedUnit 
            ? { 
                ...unit, 
                position: positionString, 
                fuel: newFuel,
                movement_used: movementCost.distance, // Устанавливаем использованное движение
                last_move_turn: getTurnData(currentTurn)?.turn_number || 0, // Устанавливаем ход движения
                // Для S и VS типов устанавливаем ограничения движения
                no_movement_turns_left: movementRestriction
              }
            : unit
        ));

        // Очищаем активные гексы
        clearActiveHexes();
        setAvailableMovementHexes([]);

        // Убираем отображение активных гексов после движения
        // Активные гексы больше не отображаются автоматически

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

  // Обработчик заправки всех кораблей
  const handleRefuelAllShips = async () => {
    if (!currentGame?.id) {
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка',
        message: 'Игра не выбрана',
        read: false
      });
      return;
    }

    try {
      const response = await refuelAPI.refuelAll({
        game_id: currentGame.id,
        fuel_amount: 4
      });

      if (response.success) {
        // Обновляем список кораблей
        if (authToken) {
          const updatedUnits = await unitsAPI.getGameUnits(currentGame.id, authToken);
          if (updatedUnits.success && updatedUnits.data) {
            setGameUnits(updatedUnits.data.units);
          }
        }

        addNotification({
          type: NotificationType.Success,
          title: 'Заправка выполнена',
          message: `Заправлено ${response.data.refueled_count} из ${response.data.total_units} кораблей (+${response.data.fuel_amount} топлива)`,
          read: false
        });
      }
    } catch (error) {
      console.error('Error refueling ships:', error);
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка',
        message: 'Не удалось заправить корабли',
        read: false
      });
    }
  };

  // Обработчик завершения фазы
  const handleCompletePhase = async () => {
    if (!currentGame?.id) return;

    try {
      await phaseAPI.nextPhase({ game_id: currentGame.id });
      
      // Обновляем информацию о текущем ходе
      const updatedTurn = await phaseAPI.getCurrentPhase(currentGame.id);
      setCurrentTurn(updatedTurn);
      
      addNotification({
        type: NotificationType.Success,
        title: 'Фаза завершена',
        message: 'Переход к следующей фазе',
        read: false
      });
    } catch (error) {
      console.error('Error completing phase:', error);
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка',
        message: 'Не удалось завершить фазу',
        read: false
      });
    }
  };

  // Обработчик клика по юниту
  const handleUnitClick = async (unitId: string, unitData: any) => {
    console.log('handleUnitClick called:', unitId, unitData);
    console.log('Current selectedUnitData before update:', selectedUnitData);
    setSelectedUnit(unitId);
    setSelectedUnitData(unitData);

    // Очищаем предыдущие активные гексы
    clearActiveHexes();

    // Проверяем, что мы находимся в фазе движения
    const currentPhase = getTurnData(currentTurn)?.current_phase;
    console.log('Current phase in handleUnitClick:', currentPhase);
    if (currentPhase !== 'movement') {
      console.log('Not in movement phase, not showing movement hexes');
      setAvailableMovementHexes([]);
      return;
    }

    // Получаем актуальную позицию юнита из gameUnits
    const currentPosition = unitData.position;
    
    // Используем обновленные данные из selectedUnitData если они есть
    const unitDataToUse = selectedUnitData && selectedUnitData.id === unitId ? selectedUnitData : unitData;
    
    // Проверяем, не двигался ли юнит уже в этом ходу (один юнит = одно движение за ход)
    const currentTurnNumber = getTurnData(currentTurn)?.turn_number || 0;
    if (unitDataToUse?.last_move_turn === currentTurnNumber) {
      console.log('Unit already moved this turn, not showing movement hexes');
      setAvailableMovementHexes([]);
      return;
    }

    // Получаем актуальные данные юнита с сервера
    let gameUnit: GameUnit | undefined;
    try {
      if (currentGame?.id && authToken) {
        const unitsResponse = await unitsAPI.getGameUnits(currentGame.id, authToken);
        if (unitsResponse.success && unitsResponse.data) {
          gameUnit = unitsResponse.data.units.find((unit: GameUnit) => unit.id === unitId);
          console.log('Fresh unit data from server:', gameUnit);
          console.log('VS Tanker server data:', {
            id: gameUnit?.id,
            name: gameUnit?.name,
            speed_rating: gameUnit?.speed_rating,
            no_movement_turns_left: gameUnit?.no_movement_turns_left,
            position: gameUnit?.position
          });
        }
      }
    } catch (error) {
      console.error('Error fetching fresh unit data:', error);
    }

    // Если не удалось получить свежие данные, используем локальные
    if (!gameUnit) {
      gameUnit = gameUnits.find(unit => unit.id === unitId);
      console.log('Using cached unit data:', gameUnit);
    }
    
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

        // Рассчитываем оставшуюся дальность движения
        // Используем speed_rating из API (приоритет) или fallback на локальные данные
        const rawSpeedType = gameUnit?.speed_rating || shipData?.speedType || 'M';
        const speedType: 'F' | 'M' | 'S' | 'VS' = (['F', 'M', 'S', 'VS'].includes(rawSpeedType) ? rawSpeedType : 'M') as 'F' | 'M' | 'S' | 'VS';
        const maxMovementRange = movementUtils.getMaxMovementDistance({ 
          speedType: speedType 
        } as any);
        const currentTurnNumber = getTurnData(currentTurn)?.turn_number || 1;
        const remainingMovement = (updatedUnitData.last_move_turn === currentTurnNumber) 
          ? Math.max(0, maxMovementRange - (updatedUnitData.movement_used || 0))
          : maxMovementRange;

        console.log('Movement calculation debug:', {
          unitName: updatedUnitData.name,
          speedType: speedType,
          gameUnitSpeedRating: gameUnit?.speed_rating,
          shipDataSpeedType: shipData?.speedType,
          maxMovementRange,
          currentTurnNumber,
          lastMoveTurn: updatedUnitData.last_move_turn,
          movementUsed: updatedUnitData.movement_used,
          remainingMovement,
          currentFuel: updatedUnitData.currentFuel
        });

        // Создаем объект с правильным speedType для расчета движения
        const movementShipData = {
          ...shipData,
          speedType: speedType
        };

        // Используем данные из unitDataToUse если они есть, иначе из gameUnit
        const noMovementTurnsLeft = unitDataToUse?.no_movement_turns_left ?? gameUnit?.no_movement_turns_left ?? 0;
        console.log('Movement restriction check:', {
          unitName: updatedUnitData.name,
          speedType: speedType,
          noMovementTurnsLeft: noMovementTurnsLeft,
          unitDataToUseNoMovement: unitDataToUse?.no_movement_turns_left,
          gameUnitNoMovement: gameUnit?.no_movement_turns_left,
          canMove: !((speedType === 'S' || speedType === 'VS') && noMovementTurnsLeft > 0)
        });

        const availableHexes = movementUtils.getAvailableMovementHexes(
          movementShipData,
          currentPosition,
          updatedUnitData.currentFuel,
          previousTurnInfo,
          remainingMovement,
          noMovementTurnsLeft
        );
        console.log('Available movement hexes calculated:', availableHexes);
        setAvailableMovementHexes(availableHexes);
      }
    } else {
      console.log('No ship data found for unit');
      setAvailableMovementHexes([]);
    }
  };

  // Обработчик клика по гексу
  const handleHexClick = (coordinate: HexCoordinate) => {
    console.log('handleHexClick called:', coordinate);
    
    // Проверяем, есть ли выбранный юнит
    if (!selectedUnit || !selectedUnitData) {
      console.log('No selected unit or unit data');
      return;
    }

    // Проверяем, что мы находимся в фазе движения
    const currentPhase = getTurnData(currentTurn)?.current_phase;
    console.log('Current phase:', currentPhase);
    if (currentPhase !== 'movement') {
      console.log('Not in movement phase');
      return;
    }

    // Проверяем, является ли гекс доступным для движения
    const isAvailableForMovement = availableMovementHexes.some(hex => 
      hex.coordinate.col === coordinate.col && 
      hex.coordinate.row === coordinate.row
    );
    console.log('Is available for movement:', isAvailableForMovement);
    console.log('Available movement hexes:', availableMovementHexes);

    if (isAvailableForMovement) {
      console.log('Executing movement to:', coordinate);
      // Выполняем движение
      handleMovement(coordinate);
    } else {
      console.log('Hex not available for movement');
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
            <div className="phase-display">
              <span className="phase-label">Текущая фаза:</span>
              <span className="phase-value">
                {(() => {
                  const turnData = getTurnData(currentTurn);
                  if (turnData && turnData.current_phase) {
                    return getPhaseDisplayName(turnData.current_phase);
                  }
                  return getPhaseDisplayName(currentGame.current_phase);
                })()}
              </span>
              <span className="phase-status">
                {(() => {
                  const turnData = getTurnData(currentTurn);
                  if (turnData && turnData.status === 'active') {
                    if (phaseTimer !== null) {
                      return `🟢 Активна (${phaseTimer}с)`;
                    }
                    return '🟢 Активна';
                  }
                  return '⚪ Ожидание';
                })()}
            </span>
            </div>
            <div className="turn-display">
              <span className="turn-label">Ход:</span>
              <span className="turn-value">
                {(() => {
                  const turnData = getTurnData(currentTurn);
                  if (turnData && turnData.turn_number !== undefined) {
                    return turnData.turn_number;
                  }
                  return currentGame.current_turn;
                })()}
            </span>
            </div>
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
                        {/* Индикатор аварийного топлива */}
                        {unit.is_emergency_fuel && (
                          <div className="emergency-fuel-indicator">
                            <span className="emergency-fuel-warning">⚠️ Аварийное топливо</span>
                            <span className="emergency-fuel-turns">
                              Осталось ходов: {unit.emergency_turn - (getTurnData(currentTurn)?.turn_number || 0)}
                            </span>
                          </div>
                        )}
              </div>
              </div>
                  ))}
              </div>
            ) : (
              <div className="no-units">Нет юнитов на карте</div>
            )}
          </div>


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
            gameUnits={gameUnits}
          />
        </div>

        {/* Правая панель - действия */}
        <div className="game-actions">
          

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
              {/* Кнопка "Начать ход 1" - только для немецкого игрока на фазе setup */}
              {(() => {
                const isGermanPlayer = currentGame?.player1_id === user?.id;
                const isGameReady = currentGame?.status === 'active' && !!currentGame?.player2_id;
                // Кнопка показывается только если активный ход - это setup фаза (turn_number = 0)
                const turnData = getTurnData(currentTurn);
                const isSetupTurn = turnData && turnData.turn_number === 0 && turnData.current_phase === 'setup';
                
                
                // Кнопка показывается только для немецкого игрока на setup фазе
                if (isGermanPlayer && isGameReady && isSetupTurn) {
                  return (
              <button 
                      className="action-button primary"
                      onClick={async () => {
                        if (!currentGame?.id) return;
                        
                        try {
                          // Начинаем первый ход
                          await phaseAPI.startTurn({ game_id: currentGame.id });
                          
                          // Обновляем информацию о текущем ходе
                          const updatedTurn = await phaseAPI.getCurrentPhase(currentGame.id);
                          setCurrentTurn(updatedTurn);
                          
                          // Уведомляем об обновлении хода
                          window.dispatchEvent(new CustomEvent('turnUpdated', { detail: updatedTurn }));
                          
                          // Обновляем информацию об игре (чтобы current_phase обновился)
                          window.dispatchEvent(new CustomEvent('gameUpdated'));
                          
                          addNotification({
                            type: NotificationType.Success,
                            title: 'Ход начат',
                            message: 'Первый ход успешно начат',
                            read: false
                          });
                        } catch (error) {
                          console.error('Ошибка начала хода:', error);
                          addNotification({
                            type: NotificationType.Error,
                            title: 'Ошибка',
                            message: 'Не удалось начать ход',
                            read: false
                          });
                        }
                      }}
                    >
                      🚀 Начать ход 1
              </button>
                  );
                }
                return null;
              })()}
              
              <button 
                className="action-button"
                onClick={handleRefuelAllShips}
                disabled={!currentGame || gameUnits.length === 0}
              >
                Заправить (+4 топлива всем кораблям)
              </button>
              <button 
                className="action-button"
                onClick={handleCompletePhase}
                disabled={!currentTurn || getTurnData(currentTurn)?.current_phase !== 'movement'}
              >
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

        {/* Основная область карты */}
        <div className="map-container">
          <HexMap
            onHexClick={handleHexClick}
            onUnitClick={handleUnitClick}
            selectedHex={selectedHex}
            playerSide={playerSide}
            availableMovementHexes={availableMovementHexes}
            activeHexes={[]}
            gameUnits={gameUnits}
          />
        </div>
      </div>

    </div>
  );
};

export default Game;
