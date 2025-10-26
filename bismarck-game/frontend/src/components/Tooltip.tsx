import React from 'react';
import './Tooltip.css';

interface TooltipProps {
  visible: boolean;
  x: number;
  y: number;
  content: {
    hexId: string;
    hexType: string;
    features: string[];
  };
}

const Tooltip: React.FC<TooltipProps> = ({ visible, x, y, content }) => {
  console.log('🔍 Tooltip render:', { visible, x, y, content });
  
  if (!visible) {
    console.log('❌ Tooltip not visible, returning null');
    return null;
  }

  console.log('🎨 Rendering tooltip with styles:', {
    position: 'fixed',
    left: x + 10,
    top: y - 10,
    zIndex: 1000,
  });

  return (
    <div 
      className="tooltip"
      style={{
        position: 'fixed',
        left: x + 10,
        top: y - 10,
        zIndex: 1000,
      }}
    >
      <div className="tooltip-content">
        <div className="tooltip-header">
          <strong>Гекс {content.hexId}</strong>
        </div>
        <div className="tooltip-body">
          <div className="tooltip-type">
            <span className="tooltip-label">Тип:</span>
            <span className={`tooltip-value tooltip-type-${content.hexType}`}>
              {getHexTypeDisplayName(content.hexType)}
            </span>
          </div>
          {content.features.length > 0 && (
            <div className="tooltip-features">
              <span className="tooltip-label">Особенности:</span>
              <ul className="tooltip-features-list">
                {content.features.map((feature, index) => (
                  <li key={index} className="tooltip-feature">
                    {getFeatureDisplayName(feature)}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// Функция для получения отображаемого имени типа гекса
const getHexTypeDisplayName = (hexType: string): string => {
  switch (hexType) {
    case 'water':
      return 'Море';
    case 'land':
      return 'Суша';
    case 'non_game':
      return 'Неигровой';
    case 'port':
      return 'Порт';
    case 'ice':
      return 'Лёд';
    case 'fog':
      return 'Туман';
    default:
      return 'Неизвестно';
  }
};

// Функция для получения отображаемого имени особенности
const getFeatureDisplayName = (feature: string): string => {
  switch (feature) {
    case 'fog':
      return 'Туман';
    case 'port':
      return 'Порт';
    case 'airport':
      return 'Аэропорт';
    case 'air_sector':
      return 'Зона действия авиации';
    case 'restricted_dd':
      return 'Зона действия немецких DD';
    case 'ice':
      return 'Ледяное поле';
    default:
      return feature;
  }
};

export default Tooltip;
