// Утилиты для работы со структурами

import { Structure, MapStructures } from '../types/editorTypes';

/**
 * Генерирует уникальный ID для структуры
 */
export function generateStructureId(): string {
  return `structure_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * Валидирует структуру
 */
export function validateStructure(structure: Structure): boolean {
  switch (structure.type) {
    case 'port':
      return structure.hexIds.length > 0 && !!structure.name;
    case 'canal':
      return !!structure.fromHex && !!structure.toHex && structure.fromHex !== structure.toHex;
    case 'convoy_route':
      return structure.hexIds.length > 0 && !!structure.name;
    case 'air_sector':
      return structure.hexIds.length > 0 && !!structure.name;
    case 'english_channel':
      return structure.hexIds.length > 0;
    case 'restricted_dd':
      return structure.hexIds.length > 0;
    default:
      return false;
  }
}

/**
 * Экспортирует структуры в JSON
 */
export function exportToJSON(structures: MapStructures): string {
  return JSON.stringify(structures, null, 2);
}

/**
 * Импортирует структуры из JSON
 */
export function importFromJSON(json: string): MapStructures {
  try {
    const data = JSON.parse(json);
    
    // Валидация структуры
    if (!data || typeof data !== 'object') {
      throw new Error('Invalid JSON structure');
    }
    
    // Проверяем наличие обязательных полей
    const requiredFields = ['ports', 'canals', 'convoyRoutes', 'airSectors'];
    for (const field of requiredFields) {
      if (!Array.isArray(data[field])) {
        data[field] = [];
      }
    }
    
    // Валидируем каждую структуру
    data.ports = data.ports.filter((s: Structure) => validateStructure(s));
    data.canals = data.canals.filter((s: Structure) => validateStructure(s));
    data.convoyRoutes = data.convoyRoutes.filter((s: Structure) => validateStructure(s));
    data.airSectors = data.airSectors.filter((s: Structure) => validateStructure(s));
    
    return data as MapStructures;
  } catch (error) {
    throw new Error(`Failed to import JSON: ${error}`);
  }
}

/**
 * Объединяет импортированные структуры с существующими
 */
export function mergeStructures(existing: MapStructures, imported: MapStructures): MapStructures {
  return {
    ports: [...existing.ports, ...imported.ports],
    canals: [...existing.canals, ...imported.canals],
    convoyRoutes: [...existing.convoyRoutes, ...imported.convoyRoutes],
    airSectors: [...existing.airSectors, ...imported.airSectors],
    englishChannel: imported.englishChannel || existing.englishChannel,
    restrictedDD: imported.restrictedDD || existing.restrictedDD,
    nonGameHexes: [...existing.nonGameHexes, ...imported.nonGameHexes],
    landAreas: [...existing.landAreas, ...imported.landAreas]
  };
}

/**
 * Скачивает JSON файл
 */
export function downloadJSON(data: string, filename: string): void {
  const blob = new Blob([data], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/**
 * Читает JSON файл
 */
export function readJSONFile(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      if (e.target?.result) {
        resolve(e.target.result as string);
      } else {
        reject(new Error('Failed to read file'));
      }
    };
    reader.onerror = reject;
    reader.readAsText(file);
  });
}

