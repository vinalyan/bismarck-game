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
import { gameEventAPI } from '../services/api/gameEventAPI';
import { searchAPI } from '../services/api/searchAPI';
import { GameTurnResponse, PHASE_NAMES } from '../types/phaseTypes';
import wsClient from '../services/websocket/websocketClient';
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

  // Определяем сторону игрока для API вызовов (строка 1077 уже определяет playerSide)
  const getPlayerSideString = (): string => {
    if (!currentGame || !user) return 'unknown';
    
    if (currentGame.player1_id === user.id) {
      return 'german'; // Player1 всегда немцы
    }
    if (currentGame.player2_id === user.id) {
      return 'allied'; // Player2 всегда союзники
    }
    
    return 'unknown';
  };

  // Генерация всех морских гексов (исключая сушу и неигровые)
  const getAllSeaHexes = (): string[] => {
    if (!mapStructures) {
      return [];
    }

    const allHexes: string[] = [];
    
    // Генерируем все гексы на карте (A1-A35, B1-B35, ..., Z1-Z35, AA1-AA35, ..., AH1-AH35)
    for (let row = 0; row < MAP_CONSTANTS.HEX_GRID_HEIGHT; row++) {
      let rowLetter: string;
      if (row < 26) {
        // A-Z
        rowLetter = String.fromCharCode(65 + row); // 65 = 'A'
      } else {
        // AA-AH
        const firstLetter = 'A';
        const secondLetter = String.fromCharCode(65 + (row - 26)); // A-H
        rowLetter = firstLetter + secondLetter;
      }
      
      for (let col = 1; col <= MAP_CONSTANTS.HEX_GRID_WIDTH; col++) {
        const hexId = `${rowLetter}${col}`;
        allHexes.push(hexId);
      }
    }

    // Фильтруем: исключаем неигровые гексы и сушу
    const seaHexes = allHexes.filter(hexId => {
      // Проверяем неигровые гексы
      if (mapStructures.nonGameHexes) {
        for (const nonGame of mapStructures.nonGameHexes) {
          if (nonGame.hexIds.includes(hexId)) {
            return false;
          }
        }
      }

      // Проверяем сушу
      if (mapStructures.landAreas) {
        for (const landArea of mapStructures.landAreas) {
          if (landArea.hexIds.includes(hexId)) {
            return false;
          }
        }
      }

      return true;
    });

    return seaHexes;
  };

  const [loadingShips] = useState(false);
  const [gameUnits, setGameUnits] = useState<GameUnit[]>([]);
  const [taskForces, setTaskForces] = useState<TaskForce[]>([]);
  const [loadingUnits, setLoadingUnits] = useState(false);
  const [searchFactorHexes, setSearchFactorHexes] = useState<Map<string, number>>(new Map());

  // Хук для управления активными гексами
  const {
    activeHexes,
    clearActiveHexes
  } = useActiveHexes();

  // Загружаем структуры карты один раз при монтировании компонента (не зависит от игры)
  useEffect(() => {
    const loadMapStructures = async () => {
      try {
        const structures = await mapService.getMapStructures();
        setMapStructures(structures);
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

    loadMapStructures();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Загружаем только один раз при монтировании

  // Загружаем данные кораблей и юнитов при монтировании компонента
  useEffect(() => {
    // Загружаем конфигурацию кораблей с бэкенда
    const loadShipsConfig = async () => {
        try {
          const ships = await shipsAPI.getAllShips();
          setShipsConfig(ships);
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

  // Расчет факторов поиска для всех морских гексов
  useEffect(() => {
    const calculateSearchFactors = async () => {
      if (!currentGame?.id || !authToken) {
        return;
      }

      if (!mapStructures) {
        return;
      }

      const turnData = getTurnData(currentTurn);
      const currentPhase = turnData?.current_phase;
      
      // Вычисляем факторы поиска только в фазах movement и search
      if (currentPhase !== 'movement' && currentPhase !== 'search') {
        setSearchFactorHexes(new Map());
        return;
      }

      const visibilityLevel = turnData?.visibility_level ?? currentGame.visibility_level ?? 1;
      
      // Получаем все морские гексы
      const seaHexes = getAllSeaHexes();
      
      if (seaHexes.length === 0) {
        return;
      }

      const playerSide = getPlayerSideString();
      if (playerSide === 'unknown') {
        return;
      }

      try {
        // Вызываем API для расчета факторов поиска
        const factors = await searchAPI.getSearchFactors(
          currentGame.id,
          seaHexes,
          playerSide as 'german' | 'allied',
          authToken
        );

        // Сохраняем результаты в Map
        const factorsMap = new Map<string, number>();
        Object.entries(factors).forEach(([hexId, factorValue]) => {
          factorsMap.set(hexId, factorValue);
        });

        setSearchFactorHexes(factorsMap);
      } catch (error) {
        // Ошибка расчета факторов поиска - просто игнорируем
      }
    };

    // Добавляем небольшую задержку для debounce, чтобы избежать множественных вызовов
    const timeoutId = setTimeout(() => {
      calculateSearchFactors();
    }, 200);

    return () => clearTimeout(timeoutId);
  }, [
    currentGame?.id,
    currentGame?.visibility_level,
    currentTurn,
    // Убираем gameUnits и taskForces из зависимостей, чтобы избежать двойных вызовов
    // Факторы поиска пересчитываются только при смене фазы/видимости
    mapStructures,
    authToken
  ]);

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

  // Подключение WebSocket при входе в игру
  useEffect(() => {
    if (currentGame?.id && authToken) {
      // Небольшая задержка перед подключением
      const timer = setTimeout(() => {
        wsClient.connect(authToken, currentGame.id).catch((error) => {
          console.error('Failed to connect WebSocket with game_id:', error);
        });
      }, 100);
      
      return () => {
        clearTimeout(timer);
        wsClient.disconnect();
      };
    }
  }, [currentGame?.id, authToken]);

  // Обработчик WebSocket событий смены фаз
  useEffect(() => {
    const handleGameEventReceived = async (event: CustomEvent) => {
      const eventData = event.detail;
      
      if (!eventData || !currentGame?.id) {
        return;
      }


      // Получаем сторону игрока для API вызовов
      const currentPlayerSide = getPlayerSideString();

      // Обрабатываем разные типы событий фаз
      switch (eventData.event) {
        case 'phase_changed':
          // Обновляем текущую фазу, события игры и юниты
          try {
            if (!authToken) {
              console.error('Auth token missing, skipping units update');
              return;
            }
            
            const results = await Promise.all([
              phaseAPI.getCurrentPhase(currentGame.id),
              gameEventAPI.getGameEvents(currentGame.id, currentPlayerSide || 'german', 15),
              unitsAPI.getGameUnits(currentGame.id, authToken)
            ]);
            
            const updatedTurn = results[0];
            const unitsResponse = results[2];
            
            if (updatedTurn) {
              setCurrentTurn(updatedTurn);
            }
            
            // Обновляем юниты и TF, особенно важно после admin фазы (сброс патрулей)
            if (unitsResponse.success && unitsResponse.data) {
              if (unitsResponse.data.units) {
                setGameUnits(unitsResponse.data.units);
              }
              if (unitsResponse.data.task_forces) {
                setTaskForces(unitsResponse.data.task_forces);
              }
            }
            
            // Показываем уведомление
            const phaseName = eventData.data?.phase ? PHASE_NAMES[eventData.data.phase as GamePhase] : 'Неизвестная фаза';
            addNotification({
              type: NotificationType.Info,
              title: 'Смена фазы',
              message: `Фаза изменена на: ${phaseName}`,
              read: false,
            });
          } catch (error) {
            console.error('Error updating phase after phase_changed event:', error);
          }
          break;

        case 'phase_advanced':
          // Обновляем текущую фазу, события игры и юниты
          try {
            if (!authToken) {
              console.error('Auth token missing, skipping units update');
              return;
            }
            
            const results = await Promise.all([
              phaseAPI.getCurrentPhase(currentGame.id),
              gameEventAPI.getGameEvents(currentGame.id, currentPlayerSide || 'german', 15),
              unitsAPI.getGameUnits(currentGame.id, authToken)
            ]);
            
            const updatedTurn = results[0];
            const unitsResponse = results[2];
            
            if (updatedTurn) {
              setCurrentTurn(updatedTurn);
            }
            
            // Обновляем юниты и TF, особенно важно после admin фазы (сброс патрулей)
            if (unitsResponse.success && unitsResponse.data) {
              if (unitsResponse.data.units) {
                setGameUnits(unitsResponse.data.units);
              }
              if (unitsResponse.data.task_forces) {
                setTaskForces(unitsResponse.data.task_forces);
              }
            }
            
            // Показываем уведомление
            const fromPhase = eventData.data?.from_phase ? PHASE_NAMES[eventData.data.from_phase as GamePhase] : 'Неизвестная фаза';
            const toPhase = eventData.data?.to_phase ? PHASE_NAMES[eventData.data.to_phase as GamePhase] : 'Неизвестная фаза';
            addNotification({
              type: NotificationType.Info,
              title: 'Переход к следующей фазе',
              message: `Переход с фазы "${fromPhase}" на "${toPhase}"`,
              read: false,
            });
          } catch (error) {
            console.error('Error updating phase after phase_advanced event:', error);
          }
          break;

        case 'turn_completed':
          // Показываем уведомление о завершении хода
          const completedTurn = eventData.data?.completed_turn || 0;
          addNotification({
            type: NotificationType.Success,
            title: 'Ход завершен',
            message: `Ход ${completedTurn} успешно завершен`,
            read: false,
          });
          
          // Обновляем текущую фазу для нового хода
          try {
            const updatedTurn = await phaseAPI.getCurrentPhase(currentGame.id);
            if (updatedTurn) {
              setCurrentTurn(updatedTurn);
            }
          } catch (error) {
            console.error('Error updating turn after turn_completed event:', error);
          }
          break;

        default:
          // Игнорируем другие события
          break;
      }
    };

    window.addEventListener('gameEventReceived', handleGameEventReceived as unknown as EventListener);
    
    return () => {
      window.removeEventListener('gameEventReceived', handleGameEventReceived as unknown as EventListener);
    };
  }, [currentGame?.id, user?.id, addNotification, getPlayerSideString]);

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
    
    if (!selectedUnitData || !currentGame?.id || !authToken) {
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

        // Обновляем данные юнитов и Task Forces с сервера после движения
        try {
          const updatedUnits = await unitsAPI.getGameUnits(currentGame.id, authToken);
          if (updatedUnits.success && updatedUnits.data) {
            // Обновляем юниты
            if (updatedUnits.data.units) {
              setGameUnits(updatedUnits.data.units);
            }
            // Обновляем Task Forces
            if (updatedUnits.data.task_forces) {
              setTaskForces(updatedUnits.data.task_forces);
            }
          }
        } catch (error) {
          console.error('Error updating units after movement:', error);
          // Fallback: локальное обновление если сервер недоступен
          const isTaskForce = selectedUnitData?.isTaskForce === true || 
                             (selectedUnitData?.name && (!selectedUnitData?.type || selectedUnitData?.type === 'taskforce'));
          
          if (isTaskForce) {
            // Обновляем Task Force
            setTaskForces(prevTaskForces => 
              prevTaskForces.map(tf => 
                tf.id === selectedUnit 
                  ? { ...tf, position: positionString }
                  : tf
              )
            );
          } else {
            // Обновляем обычный юнит
            setGameUnits(prevUnits => 
              prevUnits.map(unit => 
                unit.id === selectedUnit 
                  ? { ...unit, position: positionString, fuel: response.data?.fuel || unit.fuel }
                  : unit
              )
            );
          }
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
        // Обновляем список кораблей и Task Forces
        if (authToken) {
          const updatedUnits = await unitsAPI.getGameUnits(currentGame.id, authToken);
          if (updatedUnits.success && updatedUnits.data) {
            // Обновляем юниты
            if (updatedUnits.data.units) {
              setGameUnits(updatedUnits.data.units);
            }
            // Обновляем Task Forces
            if (updatedUnits.data.task_forces) {
              setTaskForces(updatedUnits.data.task_forces);
            }
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
      return;
    }

    // Проверяем, что мы находимся в фазе движения
    const currentPhase = getTurnData(currentTurn)?.current_phase;
    if (currentPhase !== 'movement') {
      return;
    }

    // Проверяем, является ли гекс доступным для движения
    const isAvailableForMovement = availableMovementHexes.some(hex => 
      hex.coordinate.col === coordinate.col && 
      hex.coordinate.row === coordinate.row
    );

    if (isAvailableForMovement) {
      // Выполняем движение
      handleMovement(coordinate);
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
            <div className="visibility-display">
              <span className="visibility-label">Уровень видимости:</span>
              <span className="visibility-value">
                {(() => {
                  const turnData = getTurnData(currentTurn);
                  if (turnData && turnData.visibility_level !== undefined) {
                    return turnData.visibility_level;
                  }
                  return currentGame.visibility_level !== undefined ? currentGame.visibility_level : '—';
                })()}
              </span>
            </div>
            <div className="fog-display">
              <span className="fog-label">Туман:</span>
              <span className="fog-value">
                {(() => {
                  const turnData = getTurnData(currentTurn);
                  if (turnData && turnData.is_fog !== undefined) {
                    return turnData.is_fog ? 'Да' : 'Нет';
                  }
                  return currentGame.is_fog !== undefined ? (currentGame.is_fog ? 'Да' : 'Нет') : '—';
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
            {/* Раздел Task Force */}
            {taskForces && taskForces.filter(tf => tf.position && tf.position.trim() !== '').length > 0 && (
              <div className="unit-list" style={{ marginBottom: 16 }}>
                {taskForces
                  .filter(tf => tf.position && tf.position.trim() !== '')
                  .map(tf => {
                    const memberUnits = (tf.units || [])
                      .map(uid => gameUnits.find(u => u.id === uid))
                      .filter(Boolean);
                    return (
                      <div key={tf.id} className={`unit-item ${selectedUnit === tf.id ? 'unit-selected' : ''}`}
                           onClick={() => {
                             // Клик по TF ведёт себя как по юниту
                             setSelectedUnit(tf.id);
                             setSelectedUnitData({ id: tf.id, name: tf.name, type: 'taskforce', position: tf.position, isTaskForce: true });
                           }}
                           style={{ cursor: 'pointer' }}
                      >
                        <div className="unit-header">
                          <span className="unit-name">{tf.name}</span>
                          <span className="unit-type">TF</span>
                        </div>
                        <div className="unit-status">

                        </div>

                        {memberUnits.length > 0 && (
                          <div style={{ marginTop: 8 }}>
                            {memberUnits.map((mu: any) => (
                              <div
                                key={mu!.id}
                                className={`unit-item ${selectedUnit === mu!.id ? 'unit-selected' : ''}`}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setSelectedUnit(mu!.id);
                                  // парсим позицию для совместимости с существующей логикой
                                  const pos = (mu!.position || '').toString();
                                  let coordinate: any = null;
                                  const match = pos.match(/^([A-Z]+)(\d+)$/);
                                  if (match) {
                                    const letter = match[1];
                                    const number = parseInt(match[2]);
                                    const row = letter.length === 1 ? letter.charCodeAt(0) - 65 : (letter.charCodeAt(0) - 65) * 26 + (letter.charCodeAt(1) - 65);
                                    const col = number - 1;
                                    coordinate = { letter, number, col, row };
                                  }
                                  setSelectedUnitData({
                                    id: mu!.id,
                                    type: mu!.type,
                                    side: mu!.owner || 'german',
                                    position: coordinate,
                                    name: mu!.name,
                                    maxFuel: mu!.max_fuel || 10,
                                    currentFuel: mu!.fuel || 0
                                  });
                                }}
                                style={{ cursor: 'pointer' }}
                              >
                                <div className="unit-header">
                                  <span className="unit-name">{mu!.name}</span>
                                  <span className="unit-type">{mu!.type}</span>
                                </div>
                                <div className="unit-status">
                                  <span>F: {mu!.fuel ?? 0}/{mu!.max_fuel ?? 0}</span>
                                  {(mu!.speed_rating === 'S' || mu!.speed_rating === 'VS') && mu!.no_movement_turns_left > 0 && (
                                    <span style={{ color: '#fbbf24' }}>
                                      ⏸️ Ожидание: {mu!.no_movement_turns_left} ход(ов)
                                    </span>
                                  )}
                                  {mu!.is_emergency_fuel && (
                                    <div className="emergency-fuel-indicator">
                                      <span className="emergency-fuel-warning">⚠️ Аварийное топливо</span>
                                      <span className="emergency-fuel-turns">
                                        Осталось ходов: {mu!.emergency_turn - (getTurnData(currentTurn)?.turn_number || 0)}
                                      </span>
                                      {(mu!.speed_rating === 'F' || mu!.speed_rating === 'M') && mu!.emergency_turn && (
                                        <span className="emergency-fuel-turn-info">
                                          Ход удаления: {mu!.emergency_turn}
                                        </span>
                                      )}
                                    </div>
                                  )}
                                </div>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    );
                  })}
              </div>
            )}
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
                        {/* Показываем топливо только для быстрых и средних юнитов */}
                        {(unit.speed_rating === 'F' || unit.speed_rating === 'M') && (
                          <span>F: {unit.fuel || 0}/{unit.max_fuel || 0}</span>
                        )}
                        <span>SR: {
                          unit.speed_rating === 'F' ? 'F' :
                          unit.speed_rating === 'M' ? 'M' :
                          unit.speed_rating === 'S' ? 'S' :
                          unit.speed_rating === 'VS' ? 'VS' :
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
            gameId={currentGame?.id || undefined}
            authToken={authToken}
            onRefreshData={() => {
              // Обновляем все данные игры
              if (currentGame?.id && authToken) {
                // Загружаем юниты
                unitsAPI.getGameUnits(currentGame.id, authToken).then(response => {
                  if (response.success && response.data) {
                    if (response.data.units) {
                      setGameUnits(response.data.units);
                    }
                    if (response.data.task_forces) {
                      setTaskForces(response.data.task_forces);
                    }
                  }
                });
                // Загружаем текущую фазу
                phaseAPI.getCurrentPhase(currentGame.id).then(turn => {
                  setCurrentTurn(turn);
                });
              }
            }}
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
            currentPhase={getTurnData(currentTurn)?.current_phase || 'setup'}
            searchFactorHexes={searchFactorHexes}
            visibilityLevel={getTurnData(currentTurn)?.visibility_level ?? currentGame?.visibility_level ?? 1}
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
            taskForces={taskForces}
            mapStructures={mapStructures}
            selectedUnit={selectedUnit}
            expandedStackHex={expandedStackHex}
            currentTurn={getTurnData(currentTurn)?.turn_number}
            gameId={currentGame?.id || undefined}
            authToken={authToken}
            onRefreshData={() => {
              // Обновляем все данные игры
              if (currentGame?.id && authToken) {
                // Загружаем юниты
                unitsAPI.getGameUnits(currentGame.id, authToken).then(response => {
                  if (response.success && response.data) {
                    if (response.data.units) {
                      setGameUnits(response.data.units);
                    }
                    if (response.data.task_forces) {
                      setTaskForces(response.data.task_forces);
                    }
                  }
                });
                // Загружаем текущую фазу
                phaseAPI.getCurrentPhase(currentGame.id).then(turn => {
                  setCurrentTurn(turn);
                });
              }
            }}
            onUnitStackClick={handleUnitStackClick}
            onStackedUnitSelect={handleStackedUnitSelect}
            currentPhase={getTurnData(currentTurn)?.current_phase || 'setup'}
          />
        </div>
      </div>

    </div>
  );
};

export default Game;
