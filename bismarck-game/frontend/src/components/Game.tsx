// Основной компонент игры "Погоня за Бисмарком"

import React, { useState, useEffect } from 'react';
import { useGameStore } from '../stores/gameStore';
import { ViewType, GamePhase, PlayerSide, NotificationType } from '../types/gameTypes';
import { HexCoordinate } from '../types/mapTypes';
import HexMap from './HexMap';
import './Game.css';

const Game: React.FC = () => {
  const {
    user,
    currentGame,
    logout,
    setCurrentView,
    addNotification,
    setLoading,
  } = useGameStore();

  const [selectedHex, setSelectedHex] = useState<string | null>(null);
  const [selectedUnit, setSelectedUnit] = useState<string | null>(null);
  const [showUnitInfo, setShowUnitInfo] = useState(false);

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

  // Обработчик клика по гексу
  const handleHexClick = (hexId: string) => {
    setSelectedHex(hexId);
    const hex = hexGrid.find(h => h.id === hexId);
    if (hex?.hasUnit) {
      setSelectedUnit(hexId);
      setShowUnitInfo(true);
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
          <div className="game-status">
            <h3>Статус игры</h3>
            <div className="status-item">
              <span>Фаза:</span>
              <span className="status-value">{getCurrentPhaseText(currentGame.current_phase)}</span>
            </div>
            <div className="status-item">
              <span>Ход:</span>
              <span className="status-value">{currentGame.current_turn}</span>
            </div>
            <div className="status-item">
              <span>Ваша сторона:</span>
              <span className="status-value">
                {playerSide === PlayerSide.German ? '🇩🇪 Немцы' : '🇬🇧 Союзники'}
              </span>
            </div>
          </div>

          {/* Информация о юнитах */}
          <div className="units-info">
            <h3>Ваши юниты</h3>
            <div className="unit-list">
              <div className="unit-item">
                <span className="unit-type">🚢 Линкор Бисмарк</span>
                <span className="unit-status">В море</span>
              </div>
              <div className="unit-item">
                <span className="unit-type">🚢 Тяжелый крейсер Принц Ойген</span>
                <span className="unit-status">В море</span>
              </div>
              <div className="unit-item">
                <span className="unit-type">✈️ Разведчик</span>
                <span className="unit-status">В полете</span>
              </div>
            </div>
          </div>

          {/* Выбранный гекс/юнит */}
          {selectedHex && (
            <div className="selected-info">
              <h3>Выбранная позиция</h3>
              <div className="hex-info">
                <span>Координаты: {selectedHex}</span>
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
          <div className="map-header">
            <h3>Карта Северной Атлантики</h3>
            <div className="map-legend">
              <div className="legend-item">
                <div className="legend-color water"></div>
                <span>Вода</span>
              </div>
              <div className="legend-item">
                <div className="legend-color land"></div>
                <span>Суша</span>
              </div>
              <div className="legend-item">
                <div className="legend-color german-unit"></div>
                <span>🇩🇪 Немецкие юниты</span>
              </div>
              <div className="legend-item">
                <div className="legend-color allied-unit"></div>
                <span>🇬🇧 Союзнические юниты</span>
              </div>
            </div>
          </div>
          
          <HexMap
            width={35}
            height={34}
            onHexClick={(coordinate: HexCoordinate) => {
              const hexId = `${coordinate.letter}${coordinate.number}`;
              handleHexClick(hexId);
            }}
            onHexHover={(coordinate: HexCoordinate) => {
              // Можно добавить логику подсветки при наведении
            }}
            selectedHex={selectedHex ? {
              letter: selectedHex.charAt(0),
              number: parseInt(selectedHex.slice(1)),
              q: 0, r: 0
            } : null}
            highlightedHexes={[]}
          />
        </div>

        {/* Правая панель - действия */}
        <div className="game-actions">
          <div className="action-panel">
            <h3>Действия</h3>
            <div className="action-buttons">
              <button 
                className="action-button"
                disabled={currentGame.current_phase !== GamePhase.Movement}
              >
                Движение
              </button>
              <button 
                className="action-button"
                disabled={currentGame.current_phase !== GamePhase.Search}
              >
                Поиск
              </button>
              <button 
                className="action-button"
                disabled={currentGame.current_phase !== GamePhase.Combat}
              >
                Бой
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

      {/* Модальное окно с информацией о юните */}
      {showUnitInfo && selectedUnit && (
        <div className="modal-overlay" onClick={() => setShowUnitInfo(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3>Информация о юните</h3>
              <button 
                className="modal-close"
                onClick={() => setShowUnitInfo(false)}
              >
                ×
              </button>
            </div>
            <div className="modal-body">
              <div className="unit-details">
                <div className="detail-item">
                  <span>Тип:</span>
                  <span>Линкор Бисмарк</span>
                </div>
                <div className="detail-item">
                  <span>Сторона:</span>
                  <span>🇩🇪 Немцы</span>
                </div>
                <div className="detail-item">
                  <span>Позиция:</span>
                  <span>{selectedUnit}</span>
                </div>
                <div className="detail-item">
                  <span>Топливо:</span>
                  <span>85%</span>
                </div>
                <div className="detail-item">
                  <span>Состояние:</span>
                  <span>Активен</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Game;
