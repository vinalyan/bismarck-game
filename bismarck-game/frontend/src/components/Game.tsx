// Основной компонент игры "Погоня за Бисмарком"

import React, { useState, useEffect } from 'react';
import { useGameStore } from '../stores/gameStore';
import { ViewType, GamePhase, PlayerSide, NotificationType } from '../types/gameTypes';
import { HexCoordinate } from '../types/mapTypes';
import { MovementHex } from '../utils/movementUtils';
import { useActiveHexes } from '../utils/activeHexesUtils';
import { MAP_CONSTANTS } from '../utils/hexUtils';
import { unitsAPI, GameUnit, TaskForce } from '../services/api/unitsAPI';
import { movementAPI } from '../services/api/movementAPI';
import { shipsAPI } from '../services/api/shipsAPI';
import { phaseAPI, GameTurn } from '../services/api/phaseAPI';
import { refuelAPI } from '../services/api/refuelAPI';
import { mapService, MapStructure } from '../services/api/mapService';
import { GameTurnResponse, PHASE_NAMES } from '../types/phaseTypes';
import HexMap from './HexMap';
import GameLog from './GameLog';
import './Game.css';

const Game: React.FC = () => {
  const {
    user,
    currentGame,
    authToken,
    logout,
    setCurrentView,
    addNotification,
    setShipsConfig,
    getShipsByType,
  } = useGameStore();

  const [selectedHex] = useState<HexCoordinate | null>(null);
  const [selectedUnit, setSelectedUnit] = useState<string | null>(null);
  const [selectedUnitData, setSelectedUnitData] = useState<any>(null);
  const [availableMovementHexes, setAvailableMovementHexes] = useState<MovementHex[]>([]);
  const [mapStructures, setMapStructures] = useState<MapStructure | null>(null);
  const [expandedStackHex, setExpandedStackHex] = useState<string | null>(null);
  
  // Отслеживание изменений availableMovementHexes
  useEffect(() => {
    if (availableMovementHexes.length > 0) {
      // console.log('✅ Available moves updated:', availableMovementHexes.length, 'hexes');
    }
  }, [availableMovementHexes]);
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

  const [loadingShips] = useState(false);
  const [gameUnits, setGameUnits] = useState<GameUnit[]>([]);
  const [taskForces, setTaskForces] = useState<TaskForce[]>([]);
  const [loadingUnits, setLoadingUnits] = useState(false);

  // Хук для управления активными гексами
  const {
    activeHexes,
    clearActiveHexes
  } = useActiveHexes();

  // Загружаем данные кораблей и юнитов при монтировании компонента
  useEffect(() => {
    // Загружаем конфигурацию кораблей с бэкенда
    const loadShipsConfig = async () => {
        try {
          const ships = await shipsAPI.getAllShips();
          setShipsConfig(ships);
          console.log('Ships config loaded from backend:', ships.length, 'ships');
        } catch (error) {
          console.error('Error loading ships config:', error);
          addNotification({
            type: NotificationType.Error,
            title: 'Ошибка загрузки конфигурации кораблей',
            message: 'Не удалось загрузить конфигурацию кораблей с сервера',
            read: false
          });
        }
    };

    // Загружаем структуры карты
    const loadMapStructures = async () => {
      try {
        const structures = await mapService.getMapStructures();
        setMapStructures(structures);
        console.log('Map structures loaded:', structures);
      } catch (error) {
        console.error('Error loading map structures:', error);
        addNotification({
          type: NotificationType.Error,
          title: 'Ошибка загрузки структур карты',
          message: 'Не удалось загрузить структуры карты с сервера',
          read: false
        });
      }
    };

    // Загружаем юниты игры из API
    const loadGameUnits = async () => {
      if (!currentGame?.id || !authToken) {
        return;
      }

      try {
        setLoadingUnits(true);
        const response = await unitsAPI.getGameUnits(currentGame.id, authToken);
        
        if (response.success && response.data) {
          if (response.data.units) {
            setGameUnits(response.data.units);
          }
          if (response.data.task_forces) {
            setTaskForces(response.data.task_forces);
          }
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

    loadShipsConfig();
    loadMapStructures();
    loadGameUnits();
  }, [currentGame?.id, authToken, addNotification, setShipsConfig]);

  // Автоматическое обновление юнитов каждые 5 секунд
  useEffect(() => {
    if (!currentGame?.id || !authToken) {
      return;
    }

    // Отключен автоматический polling - используем WebSocket для обновлений
    // const interval = setInterval(async () => {
    //   try {
    //     const response = await unitsAPI.getGameUnits(currentGame.id, authToken);
    //     if (response.success && response.data) {
    //       if (response.data.units) {
    //         setGameUnits(response.data.units);
    //       }
    //       if (response.data.task_forces) {
    //         setTaskForces(response.data.task_forces);
    //       }
    //     }
    //   } catch (error) {
    //     console.error('Error updating game units:', error);
    //   }
    // }, 5000); // Обновляем каждые 5 секунд

    // return () => clearInterval(interval);
  }, [currentGame?.id, authToken]);

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

  // Автоматическое обновление информации о текущей фазе каждые 10 секунд
  useEffect(() => {
    if (!currentGame?.id) {
      return;
    }

    // Отключен автоматический polling фазы - используем WebSocket для обновлений
    // const interval = setInterval(async () => {
    //   // Проверяем, что страница активна
    //   if (document.hidden) {
    //     return;
    //   }
    //   
    //   try {
    //     const turn = await phaseAPI.getCurrentPhase(currentGame.id);
    //     const previousTurn = currentTurn;
    //     
    //     setCurrentTurn(turn);
    //     
    //     // Проверяем, изменилась ли фаза
    //     if (previousTurn && turn) {
    //       const previousTurnData = getTurnData(previousTurn);
    //       const currentTurnData = getTurnData(turn);
    //       
    //       if (previousTurnData && currentTurnData) {
    //         // Если изменилась фаза или ход
    //         if (previousTurnData.current_phase !== currentTurnData.current_phase ||
    //             previousTurnData.turn_number !== currentTurnData.turn_number) {
    //           
    //           // Показываем уведомление о смене фазы
    //           if (previousTurnData.current_phase !== currentTurnData.current_phase) {
    //             addNotification({
    //               type: NotificationType.Info,
    //               title: 'Смена фазы',
    //               message: `Переход к фазе: ${PHASE_NAMES[currentTurnData.current_phase]}`,
    //               read: false
    //             });
    //           }
    //           
    //           // Показываем уведомление о новом ходе
    //           if (previousTurnData.turn_number !== currentTurnData.turn_number) {
    //             addNotification({
    //               type: NotificationType.Success,
    //               title: 'Новый ход',
    //               message: `Начат ход ${currentTurnData.turn_number}`,
    //               read: false
    //             });
    //           }
    //         }
    //       }
    //     }
    //   } catch (error) {
    //     console.error('Error updating current turn:', error);
    //   }
    // }, 10000); // Обновляем каждые 10 секунд

    // return () => clearInterval(interval);
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

    const positionString = `${targetCoordinate.letter}${targetCoordinate.number}`;

    try {
      // Отправляем запрос на движение через movementAPI
      const response = await movementAPI.moveUnit(
        currentGame.id,
        selectedUnit!,
        { unit_id: selectedUnit!, to_hex: positionString },
        authToken
      );
      
      console.log('Movement response:', response);

      if (response.success) {
        // Обновляем данные юнита с сервера
        const updatedUnitData = {
          ...selectedUnitData,
          position: targetCoordinate,
          currentFuel: response.data?.fuel || selectedUnitData.currentFuel,
          movement_used: response.data?.hexesMoved || 0,
          last_move_turn: getTurnData(currentTurn)?.turn_number || 0
        };

        setSelectedUnitData(updatedUnitData);

        // Обновляем данные юнитов с сервера после движения
        try {
          const updatedUnits = await unitsAPI.getGameUnits(currentGame.id, authToken);
          if (updatedUnits.success && updatedUnits.data && updatedUnits.data.units) {
            setGameUnits(updatedUnits.data.units);
          }
        } catch (error) {
          console.error('Error updating units after movement:', error);
          // Fallback: локальное обновление если сервер недоступен
          setGameUnits(prevUnits => 
            prevUnits.map(unit => 
              unit.id === selectedUnit 
                ? { ...unit, position: positionString, fuel: response.data?.fuel || unit.fuel }
                : unit
            )
          );
        }

        // Очищаем активные гексы
        clearActiveHexes();
        setAvailableMovementHexes([]);

        addNotification({
          type: NotificationType.Success,
          title: 'Движение выполнено',
          message: `${selectedUnitData.name} перемещен в ${positionString}`,
          read: false
        });

        console.log('Movement completed successfully');
      } else {
        addNotification({
          type: NotificationType.Error,
          title: 'Ошибка движения',
          message: response.message || 'Неизвестная ошибка',
          read: false
        });
      }
    } catch (error: any) {
      console.error('Movement error:', error);
      addNotification({
        type: NotificationType.Error,
        title: 'Ошибка движения',
        message: error.message || 'Произошла ошибка при выполнении движения',
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

  // Обработчик начала первого хода
  const handleStartFirstTurn = async () => {
    if (!currentGame?.id) return;
    
    try {
      console.log('🔄 Starting first turn...');
      
      // Начинаем первый ход
      await phaseAPI.startTurn({ game_id: currentGame.id });
      
      console.log('✅ First turn started successfully');
      
      // Обновляем информацию о текущем ходе
      const updatedTurn = await phaseAPI.getCurrentPhase(currentGame.id);
      setCurrentTurn(updatedTurn);
      
      console.log('📊 Turn data updated:', {
        turn_number: updatedTurn?.turn_number,
        current_phase: updatedTurn?.current_phase,
        status: updatedTurn?.status
      });
      
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
  };

  // Обработчик завершения фазы
  const handleCompletePhase = async () => {
    if (!currentGame?.id) return;

    try {
      
      await phaseAPI.nextPhase({ game_id: currentGame.id });
      
      
      // Обновляем информацию о текущем ходе
      const updatedTurn = await phaseAPI.getCurrentPhase(currentGame.id);
      
      // Если ход завершен (updatedTurn === null), начинаем новый ход
      if (!updatedTurn) {
        try {
          const newTurn = await phaseAPI.startTurn({ game_id: currentGame.id });
          setCurrentTurn(newTurn);
          
          
          // Уведомляем об обновлении хода
          window.dispatchEvent(new CustomEvent('turnUpdated', { detail: newTurn }));
          
          addNotification({
            type: NotificationType.Success,
            title: 'Новый ход начат',
            message: `Ход ${newTurn.turn_number} успешно начат`,
            read: false
          });
        } catch (newTurnError) {
          console.error('Error starting new turn:', newTurnError);
          addNotification({
            type: NotificationType.Error,
            title: 'Ошибка',
            message: 'Не удалось начать новый ход',
            read: false
          });
        }
      } else {
        setCurrentTurn(updatedTurn);
        
      }
      
      // Уведомление о завершении фазы (только если не начался новый ход)
      if (updatedTurn) {
        addNotification({
          type: NotificationType.Success,
          title: 'Фаза завершена',
          message: 'Переход к следующей фазе',
          read: false
        });
      }
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

  // Обработчик клика по стеку юнитов
  const handleUnitStackClick = (hexId: string, units: any[]) => {
    console.log('Unit stack clicked:', hexId, units);
    setExpandedStackHex(hexId);
  };

  // Обработчик выбора юнита из стека
  const handleStackedUnitSelect = async (unit: any) => {
    console.log('Stacked unit selected:', unit);
    
    // Если кликнули на уже выбранный юнит - сбрасываем выбор
    if (selectedUnit === unit.id) {
      setSelectedUnit(null);
      setSelectedUnitData(null);
      setAvailableMovementHexes([]);
      clearActiveHexes();
      setExpandedStackHex(null);
      return;
    }
    
    // Сворачиваем стек
    setExpandedStackHex(null);
    // Выбираем юнит
    setSelectedUnit(unit.id);
    setSelectedUnitData(unit);
    
    // Очищаем предыдущие активные гексы
    clearActiveHexes();

    // Проверяем, что мы находимся в фазе движения
    const currentPhase = getTurnData(currentTurn)?.current_phase;
    if (currentPhase !== 'movement') {
      setAvailableMovementHexes([]);
      return;
    }

    // Получаем актуальную позицию юнита
    const currentPosition = unit.position;
    
    // Проверяем, не двигался ли юнит уже в этом ходу
    const currentTurnNumber = getTurnData(currentTurn)?.turn_number || 0;
    if (unit.last_move_turn === currentTurnNumber) {
      setAvailableMovementHexes([]);
      return;
    }

    // Получаем актуальные данные юнита с сервера
    let gameUnit: GameUnit | undefined;
    try {
      if (currentGame?.id && authToken) {
        const unitsResponse = await unitsAPI.getGameUnits(currentGame.id, authToken);
        if (unitsResponse.success && unitsResponse.data) {
          gameUnit = unitsResponse.data.units.find((unit: GameUnit) => unit.id === unit.id);
        }
      }
    } catch (error) {
      console.error('Error fetching fresh unit data:', error);
    }

    // Если не удалось получить свежие данные, используем локальные
    if (!gameUnit) {
      gameUnit = gameUnits.find(u => u.id === unit.id);
    }
    
    // Получаем конфигурацию корабля
    const shipsByType = getShipsByType(gameUnit?.type || unit.type);
    const shipConfig = shipsByType.length > 0 ? shipsByType[0] : null;

    if (shipConfig) {
      const updatedUnitData = {
        ...unit,
        ...gameUnit, // Добавляем данные из API
        position: currentPosition,
        shipConfig: shipConfig,
        maxFuel: shipConfig.maxFuel,
        currentFuel: gameUnit?.fuel || unit.fuel || Math.floor(shipConfig.maxFuel * 0.85)
      };
      setSelectedUnitData(updatedUnitData);

      // Получаем доступные гексы для движения с сервера
      if (currentPosition && currentGame?.id && authToken) {
        try {
          const availableMovesResponse = await movementAPI.getAvailableMoves(currentGame.id, unit.id, authToken);
          
          if (availableMovesResponse && availableMovesResponse.available_hexes) {
            // Преобразуем ответ сервера в формат MovementHex[]
            const availableHexes: MovementHex[] = availableMovesResponse.available_hexes.map(hex => {
              // Парсим строку гекса (например, "K15") в HexCoordinate
              const match = hex.match(/^([A-Z]+)(\d+)$/);
              if (!match) {
                console.error('Invalid hex format:', hex);
                return null;
              }
              
              const letter = match[1];
              const number = parseInt(match[2]);
              
              // Преобразуем в координаты сетки
              const row = letter.length === 1 ? letter.charCodeAt(0) - 65 : (letter.charCodeAt(0) - 65) * 26 + (letter.charCodeAt(1) - 65);
              const col = number - 1;
              
              return {
                coordinate: {
                  letter,
                  number,
                  col,
                  row
                },
                distance: 1, // Временное значение
                fuelCost: 1, // Временное значение
                isReachable: true // Временное значение
              };
            }).filter(hex => hex !== null) as MovementHex[];
            
            setAvailableMovementHexes(availableHexes);
          } else {
            setAvailableMovementHexes([]);
          }
        } catch (error) {
          console.error('Error fetching available moves:', error);
          setAvailableMovementHexes([]);
        }
      }
    }
  };

  // Обработчик клика по юниту
  const handleUnitClick = async (unitId: string, unitData: any) => {
    const isTaskForce = unitData.isTaskForce === true || (unitData.name && (!unitData.type || unitData.type === 'taskforce'));
    console.log('🎯 Unit clicked:', unitId, 'data:', {
      name: unitData.name,
      type: unitData.type,
      isTaskForce: isTaskForce,
      hasPosition: !!unitData.position
    });
    
    // Если кликнули на уже выбранный юнит - сбрасываем выбор
    if (selectedUnit === unitId) {
      setSelectedUnit(null);
      setSelectedUnitData(null);
      setAvailableMovementHexes([]);
      clearActiveHexes();
      return;
    }
    
    setSelectedUnit(unitId);
    setSelectedUnitData(unitData);

    // Очищаем предыдущие активные гексы
    clearActiveHexes();

    // Проверяем, что мы находимся в фазе движения
    const currentPhase = getTurnData(currentTurn)?.current_phase;
    if (currentPhase !== 'movement') {
      // console.log('❌ Not in movement phase');
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
      // console.log('❌ Unit already moved this turn');
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
        }
      }
    } catch (error) {
      console.error('Error fetching fresh unit data:', error);
    }

    // Если не удалось получить свежие данные, используем локальные
    if (!gameUnit) {
      gameUnit = gameUnits.find(unit => unit.id === unitId);
    }
    
    // isTaskForce уже определена в начале функции
    
    // Получаем конфигурацию корабля из store по типу (для Task Force пропускаем)
    const shipsByType = isTaskForce ? [] : getShipsByType(gameUnit?.type || unitData.type);
    const shipConfig = shipsByType.length > 0 ? shipsByType[0] : null;

    // Обновляем данные юнита с информацией о корабле
    if (shipConfig || isTaskForce) {
      const updatedUnitData = {
        ...unitData,
        ...gameUnit, // Добавляем данные из API
        position: currentPosition, // Используем актуальную позицию
        shipConfig: shipConfig,
        maxFuel: isTaskForce ? 100 : (shipConfig?.maxFuel || 100), // Заглушка для Task Force
        currentFuel: isTaskForce ? 85 : (gameUnit?.fuel || unitData.fuel || Math.floor((shipConfig?.maxFuel || 100) * 0.85)) // Заглушка для Task Force
      };
      setSelectedUnitData(updatedUnitData);

      // Получаем доступные гексы для движения с сервера
      if (currentPosition && currentGame?.id && authToken) {
        try {
          console.log('🚀 Loading available moves for unit:', unitId, 'type:', unitData.type || unitData.name);
          const availableMovesResponse = await movementAPI.getAvailableMoves(currentGame.id, unitId, authToken);
          // console.log('🎯 Server response - available_hexes:', availableMovesResponse.available_hexes);
          // console.log('🎯 Server response - max_distance:', availableMovesResponse.max_distance);
          
          if (availableMovesResponse && availableMovesResponse.available_hexes) {
            // Преобразуем ответ сервера в формат MovementHex[]
            const availableHexes: MovementHex[] = availableMovesResponse.available_hexes.map(hex => {
              // Парсим строку гекса (например, "K15") в HexCoordinate
              const match = hex.match(/^([A-Z]+)(\d+)$/);
              if (!match) {
                console.error('Invalid hex format:', hex);
                return null;
              }
              
              const letter = match[1];
              const number = parseInt(match[2]);
              
              // Преобразуем в координаты сетки
              const row = letter.length === 1 ? letter.charCodeAt(0) - 65 : (letter.charCodeAt(0) - 65) * 26 + (letter.charCodeAt(1) - 65);
              const col = number - 1;
              
              return {
                coordinate: {
                  letter,
                  number,
                  col,
                  row
                },
                fuelCost: availableMovesResponse.fuel_costs?.[hex] || 0,
                distance: 1, // Будет рассчитано позже
                isReachable: true
              };
            }).filter(hex => hex !== null) as MovementHex[];
            
            // console.log('✅ Found', availableHexes.length, 'available moves');
            // console.log('🎯 Available hexes:', availableHexes.map(h => `${h.coordinate.letter}${h.coordinate.number}`));
            setAvailableMovementHexes(availableHexes);
          } else {
            // console.log('❌ No available moves from server');
            setAvailableMovementHexes([]);
          }
        } catch (error) {
          console.error('Error fetching available moves from server:', error);
          addNotification({
            type: NotificationType.Error,
            title: 'Ошибка получения доступных ходов',
            message: 'Не удалось получить доступные ходы с сервера',
            read: false
          });
          setAvailableMovementHexes([]);
        }
      }
    } else {
      console.log('No ship config found for unit and not a Task Force');
      setAvailableMovementHexes([]);
    }
  };

  // Локальная функция расчета удалена - используем только бэкенд

  // Обработчик клика по гексу
  const handleHexClick = async (coordinate: HexCoordinate) => {
    // Сворачиваем стек при клике на другой гекс
    if (expandedStackHex) {
      setExpandedStackHex(null);
    }

    // Снимаем выделение с юнитов при клике на любой гекс
    if (selectedUnit) {
      setSelectedUnit(null);
      setSelectedUnitData(null);
      setAvailableMovementHexes([]);
      clearActiveHexes();
    }

    // Если нет выбранного юнита, просто выходим
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
                    return PHASE_NAMES[turnData.current_phase];
                  }
                  return PHASE_NAMES[currentGame.current_phase as GamePhase] || currentGame.current_phase;
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
                  .map((unit) => {
                    // Проверяем, может ли юнит двигаться
                    const currentPhase = getTurnData(currentTurn)?.current_phase;
                    const currentTurnNumber = getTurnData(currentTurn)?.turn_number || 0;
                    const canMove = currentPhase === 'movement' && 
                                   unit.last_move_turn !== currentTurnNumber &&
                                   unit.fuel && unit.fuel > 0;
                    
                    return (
                    <div 
                      key={unit.id} 
                      className={`unit-item ${!canMove ? 'unit-disabled' : ''} ${selectedUnit === unit.id ? 'unit-selected' : ''}`}
                      onClick={() => {
                        // Парсим позицию юнита
                        const positionMatch = unit.position.match(/^([A-Z]+)(\d+)$/);
                        if (positionMatch) {
                          const letter = positionMatch[1];
                          const number = parseInt(positionMatch[2]);
                          const row = letter.length === 1 ? letter.charCodeAt(0) - 65 : (letter.charCodeAt(0) - 65) * 26 + (letter.charCodeAt(1) - 65);
                          const col = number - 1;
                          
                          const coordinate: HexCoordinate = {
                            letter,
                            number,
                            col,
                            row
                          };
                          
                          // Вызываем handleUnitClick с данными юнита
                          handleUnitClick(unit.id, {
                            id: unit.id,
                            type: unit.type,
                            side: unit.owner || 'german',
                            position: coordinate,
                            name: unit.name,
                            maxFuel: unit.max_fuel || 10,
                            currentFuel: unit.fuel || 8
                          });
                        }
                      }}
                      style={{ cursor: 'pointer' }}
                    >
                      <div className="unit-header">
                        <span className="unit-name">{unit.name}</span>
                        <span className="unit-type">{unit.type}</span>
                      </div>
                      <div className="unit-status">
                        <span>Позиция: {unit.position}</span>
                        {/* Показываем топливо только для быстрых и средних юнитов */}
                        {(unit.speed_rating === 'F' || unit.speed_rating === 'M') && (
                          <span>Топливо: {unit.fuel || 0}/{unit.max_fuel || 0}</span>
                        )}
                        <span>Скорость: {
                          unit.speed_rating === 'F' ? 'Быстрый' :
                          unit.speed_rating === 'M' ? 'Средний' :
                          unit.speed_rating === 'S' ? 'Медленный' :
                          unit.speed_rating === 'VS' ? 'Очень медленный' :
                          unit.speed_rating || 'Неизвестно'
                        }</span>
                        {/* Информация о ограничениях движения для медленных юнитов */}
                        {(unit.speed_rating === 'S' || unit.speed_rating === 'VS') && unit.no_movement_turns_left > 0 && (
                          <span style={{ color: '#fbbf24' }}>
                            ⏸️ Ожидание: {unit.no_movement_turns_left} ход(ов)
                          </span>
                        )}
                        {/* Индикатор аварийного топлива */}
                        {unit.is_emergency_fuel && (
                          <div className="emergency-fuel-indicator">
                            <span className="emergency-fuel-warning">⚠️ Аварийное топливо</span>
                            <span className="emergency-fuel-turns">
                              Осталось ходов: {unit.emergency_turn - (getTurnData(currentTurn)?.turn_number || 0)}
                            </span>
                            {/* Информация об emergency_turn для быстрых юнитов */}
                            {(unit.speed_rating === 'F' || unit.speed_rating === 'M') && unit.emergency_turn && (
                              <span className="emergency-fuel-turn-info">
                                Ход удаления: {unit.emergency_turn}
                              </span>
                            )}
                          </div>
                        )}
              </div>
              </div>
                    );
                  })}
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
        <div 
          className="game-map"
          onClick={(e) => {
            // Снимаем выделение при клике на пустую область карты
            if (e.target === e.currentTarget) {
              if (selectedUnit) {
                setSelectedUnit(null);
                setSelectedUnitData(null);
                setAvailableMovementHexes([]);
                clearActiveHexes();
              }
              if (expandedStackHex) {
                setExpandedStackHex(null);
              }
            }
          }}
        >
          
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
            taskForces={taskForces}
            selectedUnit={selectedUnit}
            expandedStackHex={expandedStackHex}
            currentTurn={getTurnData(currentTurn)?.turn_number}
            onUnitStackClick={handleUnitStackClick}
            onStackedUnitSelect={handleStackedUnitSelect}
            onRefuelAllShips={handleRefuelAllShips}
            onCompletePhase={handleCompletePhase}
            onStartFirstTurn={handleStartFirstTurn}
            isRefuelDisabled={!currentGame || gameUnits.length === 0}
            isCompletePhaseDisabled={!currentTurn || getTurnData(currentTurn)?.current_phase !== 'movement'}
            isStartFirstTurnVisible={(() => {
              const isGermanPlayer = currentGame?.player1_id === user?.id;
              const isGameReady = currentGame?.status === 'active' && !!currentGame?.player2_id;
              const turnData = getTurnData(currentTurn);
              const isSetupTurn = turnData && turnData.turn_number === 0 && turnData.current_phase === 'setup';
              return !!(isGermanPlayer && isGameReady && isSetupTurn);
            })()}
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
              {/* Кнопки управления игрой перенесены в HexMap */}
            </div>
          </div>

          {/* Лог игры */}
          {currentGame && (
            <GameLog gameId={currentGame.id} />
          )}
        </div>

        {/* Основная область карты */}
        <div 
          className="map-container"
          onClick={(e) => {
            // Снимаем выделение при клике на пустую область карты
            if (e.target === e.currentTarget) {
              if (selectedUnit) {
                setSelectedUnit(null);
                setSelectedUnitData(null);
                setAvailableMovementHexes([]);
                clearActiveHexes();
              }
              if (expandedStackHex) {
                setExpandedStackHex(null);
              }
            }
          }}
        >
          <HexMap
            onHexClick={handleHexClick}
            onUnitClick={handleUnitClick}
            selectedHex={selectedHex}
            playerSide={playerSide}
            availableMovementHexes={availableMovementHexes}
            activeHexes={[]}
            gameUnits={gameUnits}
            mapStructures={mapStructures}
            selectedUnit={selectedUnit}
            expandedStackHex={expandedStackHex}
            currentTurn={getTurnData(currentTurn)?.turn_number}
            onUnitStackClick={handleUnitStackClick}
            onStackedUnitSelect={handleStackedUnitSelect}
          />
        </div>
      </div>

    </div>
  );
};

export default Game;
