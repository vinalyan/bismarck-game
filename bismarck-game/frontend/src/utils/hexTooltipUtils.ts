import { MapStructure, HexFeature } from '../types/mapTypes';

/**
 * Определяет особенности гекса на основе его ID и структур карты
 */
export const getHexFeatures = (hexId: string, mapStructures: MapStructure | null): HexFeature[] => {
  if (!mapStructures) return [];

  const features: HexFeature[] = [];

  // Проверяем зону ограничения для немецких DD
  if (mapStructures.restrictedDD && mapStructures.restrictedDD.hexIds.includes(hexId)) {
    features.push('restricted_dd');
  }

  // TODO: Добавить проверки для других особенностей когда они будут реализованы
  // - fog: туман
  // - port: порт
  // - airport: аэропорт
  // - air_sector: зона действия авиации
  // - ice: ледяное поле

  return features;
};

/**
 * Определяет тип гекса для отображения в подсказке
 */
export const getHexTypeForTooltip = (hexId: string, mapStructures: MapStructure | null): string => {
  if (!mapStructures) return 'water';

  // Проверяем неигровые гексы
  for (const nonGameArea of mapStructures.nonGameHexes) {
    if (nonGameArea.hexIds.includes(hexId)) {
      return 'non_game';
    }
  }

  // Проверяем сухопутные гексы
  for (const landArea of mapStructures.landAreas) {
    if (landArea.hexIds.includes(hexId)) {
      return 'land';
    }
  }

  // По умолчанию - морской гекс
  return 'water';
};

/**
 * Создает данные для подсказки гекса
 */
export const createHexTooltip = (hexId: string, mapStructures: MapStructure | null) => {
  const hexType = getHexTypeForTooltip(hexId, mapStructures);
  const features = getHexFeatures(hexId, mapStructures);
  
  return {
    hexId,
    hexType,
    features,
  };
};
