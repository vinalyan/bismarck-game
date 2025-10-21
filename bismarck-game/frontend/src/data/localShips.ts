// Локальные данные кораблей для работы без бэкенда

export interface ShipData {
  id: string;
  name: string;
  type: string;
  side: 'german' | 'allied';
  maxFuel: number;
  baseEvasion: number;
  radarLevel: number;
  hullBoxes: number;
  basePrimaryArmamentBow: number;
  basePrimaryArmamentStern: number;
  baseSecondaryArmament: number;
  maxTorpedos: number;
  speedType: 'F' | 'M' | 'S' | 'VS';
  setupHex?: string; // Стартовая позиция при начале игры
  notes?: string;
  specialRules?: SpecialRule[];
}

export interface SpecialRule {
  type: string;
  description: string;
  isActive: boolean;
}

export const LOCAL_SHIPS_DATA: ShipData[] = [
  {
    id: "bismarck",
    name: "BISMARCK",
    type: "BB",
    side: "german",
    maxFuel: 18,
    baseEvasion: 30,
    radarLevel: 2,
    hullBoxes: 12,
    basePrimaryArmamentBow: 8,
    basePrimaryArmamentStern: 2,
    baseSecondaryArmament: 8,
    maxTorpedos: 0,
    speedType: "F",
    setupHex: "J30",
    notes: "После первого раунда боя считается кораблем без радара",
    specialRules: [
      {
        type: "radar_loss_after_first_round",
        description: "После первого раунда боя считается кораблем без радара",
        isActive: true
      }
    ]
  },
  {
    id: "scharnhorst",
    name: "SCHARNHORST",
    type: "BC",
    side: "german",
    maxFuel: 18,
    baseEvasion: 31,
    radarLevel: 2,
    hullBoxes: 9,
    basePrimaryArmamentBow: 6,
    basePrimaryArmamentStern: 3,
    baseSecondaryArmament: 2,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "gneisenau",
    name: "GNEISENAU",
    type: "BC",
    side: "german",
    maxFuel: 18,
    baseEvasion: 31,
    radarLevel: 2,
    hullBoxes: 9,
    basePrimaryArmamentBow: 6,
    basePrimaryArmamentStern: 3,
    baseSecondaryArmament: 2,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "prinz_eugen",
    name: "PRINZ EUGEN",
    type: "CA",
    side: "german",
    maxFuel: 9,
    baseEvasion: 32,
    radarLevel: 2,
    hullBoxes: 5,
    basePrimaryArmamentBow: 3,
    basePrimaryArmamentStern: 3,
    baseSecondaryArmament: 5,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "koln",
    name: "KÖLN",
    type: "CL",
    side: "german",
    maxFuel: 12,
    baseEvasion: 33,
    radarLevel: 2,
    hullBoxes: 12,
    basePrimaryArmamentBow: 2,
    basePrimaryArmamentStern: 3,
    baseSecondaryArmament: 0,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "5_zerstorerfl",
    name: "5. ZERSTÖRERFL.",
    type: "DD",
    side: "german",
    maxFuel: 4,
    baseEvasion: 35,
    radarLevel: 2,
    hullBoxes: 8,
    basePrimaryArmamentBow: 3,
    basePrimaryArmamentStern: 3,
    baseSecondaryArmament: 0,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "6_zerstorerfl",
    name: "6. ZERSTÖRERFL",
    type: "DD",
    side: "german",
    maxFuel: 4,
    baseEvasion: 35,
    radarLevel: 2,
    hullBoxes: 8,
    basePrimaryArmamentBow: 3,
    basePrimaryArmamentStern: 3,
    baseSecondaryArmament: 0,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "prince_of_wales",
    name: "P. OF WALES",
    type: "BB",
    side: "allied",
    maxFuel: 12,
    baseEvasion: 27,
    radarLevel: 2,
    hullBoxes: 9,
    basePrimaryArmamentBow: 9,
    basePrimaryArmamentStern: 5,
    baseSecondaryArmament: 3,
    maxTorpedos: 0,
    speedType: "F",
    notes: "Имеет ненадежное главное вооружение",
    specialRules: [
      {
        type: "unreliable_main_armament",
        description: "Ненадежное главное вооружение - особенности работы отражены в правилах игры",
        isActive: true
      }
    ]
  },
  {
    id: "king_george_v",
    name: "KING GEORGE V",
    type: "BB",
    side: "allied",
    maxFuel: 12,
    baseEvasion: 28,
    radarLevel: 2,
    hullBoxes: 9,
    basePrimaryArmamentBow: 9,
    basePrimaryArmamentStern: 5,
    baseSecondaryArmament: 3,
    maxTorpedos: 0,
    speedType: "M",
    notes: "Имеет ненадежное главное вооружение",
    specialRules: [
      {
        type: "unreliable_main_armament",
        description: "Ненадежное главное вооружение - особенности работы отражены в правилах игры",
        isActive: true
      }
    ]
  },
  {
    id: "rodney",
    name: "RODNEY",
    type: "BB",
    side: "allied",
    maxFuel: 18,
    baseEvasion: 22,
    radarLevel: 1,
    hullBoxes: 9,
    basePrimaryArmamentBow: 9,
    basePrimaryArmamentStern: 5,
    baseSecondaryArmament: 3,
    maxTorpedos: 9,
    speedType: "S",
    notes: "Главный калибр в корме может стрелять только в начальной фазе боя",
    specialRules: [
      {
        type: "stern_guns_initial_phase_only",
        description: "Главный калибр в корме может стрелять только в начальной фазе боя",
        isActive: true
      }
    ]
  },
  {
    id: "nelson",
    name: "NELSON",
    type: "BB",
    side: "allied",
    maxFuel: 18,
    baseEvasion: 22,
    radarLevel: 2,
    hullBoxes: 9,
    basePrimaryArmamentBow: 9,
    basePrimaryArmamentStern: 5,
    baseSecondaryArmament: 3,
    maxTorpedos: 9,
    speedType: "S",
    notes: "Главный калибр в корме может стрелять только в начальной фазе боя",
    specialRules: [
      {
        type: "stern_guns_initial_phase_only",
        description: "Главный калибр в корме может стрелять только в начальной фазе боя",
        isActive: true
      }
    ]
  },
  {
    id: "ramillies",
    name: "RAMILLIES",
    type: "BB",
    side: "allied",
    maxFuel: 7,
    baseEvasion: 20,
    radarLevel: 0,
    hullBoxes: 7,
    basePrimaryArmamentBow: 7,
    basePrimaryArmamentStern: 7,
    baseSecondaryArmament: 2,
    maxTorpedos: 0,
    speedType: "VS"
  },
  {
    id: "revenge",
    name: "REVENGE",
    type: "BB",
    side: "allied",
    maxFuel: 7,
    baseEvasion: 21,
    radarLevel: 1,
    hullBoxes: 7,
    basePrimaryArmamentBow: 7,
    basePrimaryArmamentStern: 7,
    baseSecondaryArmament: 2,
    maxTorpedos: 0,
    speedType: "VS"
  },
  {
    id: "ark_royal",
    name: "ARK ROYAL",
    type: "CV",
    side: "allied",
    maxFuel: 17,
    baseEvasion: 31,
    radarLevel: 0,
    hullBoxes: 5,
    basePrimaryArmamentBow: 1,
    basePrimaryArmamentStern: 0,
    baseSecondaryArmament: 5,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "victorious",
    name: "VICTORIOUS",
    type: "CV",
    side: "allied",
    maxFuel: 15,
    baseEvasion: 30,
    radarLevel: 2,
    hullBoxes: 6,
    basePrimaryArmamentBow: 2,
    basePrimaryArmamentStern: 0,
    baseSecondaryArmament: 6,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "repulse",
    name: "REPULSE",
    type: "BC",
    side: "allied",
    maxFuel: 8,
    baseEvasion: 30,
    radarLevel: 0,
    hullBoxes: 6,
    basePrimaryArmamentBow: 7,
    basePrimaryArmamentStern: 3,
    baseSecondaryArmament: 1,
    maxTorpedos: 4,
    speedType: "F"
  },
  {
    id: "hood",
    name: "HOOD",
    type: "BC",
    side: "allied",
    maxFuel: 20,
    baseEvasion: 28,
    radarLevel: 2,
    hullBoxes: 8,
    basePrimaryArmamentBow: 7,
    basePrimaryArmamentStern: 7,
    baseSecondaryArmament: 1,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "renown",
    name: "RENOWN",
    type: "BC",
    side: "allied",
    maxFuel: 8,
    baseEvasion: 30,
    radarLevel: 2,
    hullBoxes: 6,
    basePrimaryArmamentBow: 7,
    basePrimaryArmamentStern: 3,
    baseSecondaryArmament: 2,
    maxTorpedos: 4,
    speedType: "F"
  },
  {
    id: "suffolk",
    name: "SUFFOLK",
    type: "CA",
    side: "allied",
    maxFuel: 10,
    baseEvasion: 31,
    radarLevel: 2,
    hullBoxes: 4,
    basePrimaryArmamentBow: 2,
    basePrimaryArmamentStern: 2,
    baseSecondaryArmament: 4,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "london",
    name: "LONDON",
    type: "CA",
    side: "allied",
    maxFuel: 10,
    baseEvasion: 31,
    radarLevel: 2,
    hullBoxes: 4,
    basePrimaryArmamentBow: 2,
    basePrimaryArmamentStern: 2,
    baseSecondaryArmament: 4,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "norfolk",
    name: "NORFOLK",
    type: "CA",
    side: "allied",
    maxFuel: 10,
    baseEvasion: 31,
    radarLevel: 1,
    hullBoxes: 4,
    basePrimaryArmamentBow: 2,
    basePrimaryArmamentStern: 2,
    baseSecondaryArmament: 4,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "dorsetshire",
    name: "DORSETSHIRE",
    type: "CA",
    side: "allied",
    maxFuel: 10,
    baseEvasion: 31,
    radarLevel: 0,
    hullBoxes: 4,
    basePrimaryArmamentBow: 2,
    basePrimaryArmamentStern: 2,
    baseSecondaryArmament: 4,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "exeter",
    name: "EXETER",
    type: "CA",
    side: "allied",
    maxFuel: 13,
    baseEvasion: 32,
    radarLevel: 1,
    hullBoxes: 3,
    basePrimaryArmamentBow: 2,
    basePrimaryArmamentStern: 1,
    baseSecondaryArmament: 3,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "weissenburg",
    name: "WEISSENBURG",
    type: "TK",
    side: "german",
    maxFuel: 15,
    baseEvasion: 12,
    radarLevel: 0,
    hullBoxes: 6,
    basePrimaryArmamentBow: 0,
    basePrimaryArmamentStern: 0,
    baseSecondaryArmament: 0,
    maxTorpedos: 0,
    speedType: "VS"
  },
  {
    id: "belchen",
    name: "BELCHEN",
    type: "TK",
    side: "german",
    maxFuel: 15,
    baseEvasion: 12,
    radarLevel: 0,
    hullBoxes: 6,
    basePrimaryArmamentBow: 0,
    basePrimaryArmamentStern: 0,
    baseSecondaryArmament: 0,
    maxTorpedos: 0,
    speedType: "VS"
  },
  {
    id: "lotheringen",
    name: "LOTHERINGEN",
    type: "TK",
    side: "german",
    maxFuel: 15,
    baseEvasion: 12,
    radarLevel: 0,
    hullBoxes: 6,
    basePrimaryArmamentBow: 0,
    basePrimaryArmamentStern: 0,
    baseSecondaryArmament: 0,
    maxTorpedos: 0,
    speedType: "VS"
  },
  {
    id: "gonzenheim",
    name: "GONZENHEIM",
    type: "TK",
    side: "german",
    maxFuel: 15,
    baseEvasion: 9,
    radarLevel: 0,
    hullBoxes: 6,
    basePrimaryArmamentBow: 0,
    basePrimaryArmamentStern: 0,
    baseSecondaryArmament: 0,
    maxTorpedos: 0,
    speedType: "VS"
  },
  {
    id: "kota_pinang",
    name: "KOTA PINANG",
    type: "TK",
    side: "german",
    maxFuel: 15,
    baseEvasion: 14,
    radarLevel: 0,
    hullBoxes: 6,
    basePrimaryArmamentBow: 0,
    basePrimaryArmamentStern: 0,
    baseSecondaryArmament: 0,
    maxTorpedos: 0,
    speedType: "VS"
  },
  {
    id: "test_cruiser_m",
    name: "TEST CRUISER M",
    type: "CA",
    side: "allied",
    maxFuel: 8,
    baseEvasion: 20,
    radarLevel: 1,
    hullBoxes: 6,
    setupHex: "B15",
    basePrimaryArmamentBow: 4,
    basePrimaryArmamentStern: 4,
    baseSecondaryArmament: 6,
    maxTorpedos: 0,
    speedType: "M",
    notes: "Тестовый крейсер класса M"
  },
  {
    id: "test_destroyer_m",
    name: "TEST DESTROYER M",
    type: "DD", 
    side: "german",
    maxFuel: 6,
    baseEvasion: 15,
    radarLevel: 0,
    hullBoxes: 3,
    setupHex: "I25",
    basePrimaryArmamentBow: 2,
    basePrimaryArmamentStern: 2,
    baseSecondaryArmament: 4,
    maxTorpedos: 8,
    speedType: "M",
    notes: "Тестовый эсминец класса M"
  },
  {
    id: "test_cruiser_s",
    name: "TEST CRUISER S",
    type: "CA",
    side: "allied",
    maxFuel: 10,
    baseEvasion: 18,
    radarLevel: 1,
    hullBoxes: 8,
    setupHex: "C10",
    basePrimaryArmamentBow: 6,
    basePrimaryArmamentStern: 6,
    baseSecondaryArmament: 8,
    maxTorpedos: 0,
    speedType: "S",
    notes: "Тестовый крейсер класса S (медленный)"
  },
  {
    id: "test_tanker_vs",
    name: "TEST TANKER VS",
    type: "TK",
    side: "german",
    maxFuel: 25,
    baseEvasion: 5,
    radarLevel: 0,
    hullBoxes: 12,
    setupHex: "H20",
    basePrimaryArmamentBow: 0,
    basePrimaryArmamentStern: 0,
    baseSecondaryArmament: 2,
    maxTorpedos: 0,
    speedType: "VS",
    notes: "Тестовый танкер класса VS (очень медленный)"
  }
];

// Утилиты для работы с локальными данными
export const localShipsUtils = {
  // Получить корабль по ID
  getShipById: (id: string): ShipData | undefined => {
    return LOCAL_SHIPS_DATA.find(ship => ship.id === id);
  },

  // Получить корабли по стороне
  getShipsBySide: (side: 'german' | 'allied'): ShipData[] => {
    return LOCAL_SHIPS_DATA.filter(ship => ship.side === side);
  },

  // Получить корабли по типу
  getShipsByType: (type: string): ShipData[] => {
    return LOCAL_SHIPS_DATA.filter(ship => ship.type === type);
  },

  // Получить корабли по классу скорости
  getShipsBySpeedType: (speedType: string): ShipData[] => {
    return LOCAL_SHIPS_DATA.filter(ship => ship.speedType === speedType);
  },

  // Получить все уникальные типы кораблей
  getAllShipTypes: (): string[] => {
    const types = LOCAL_SHIPS_DATA.map(ship => ship.type);
    return types.filter((type, index) => types.indexOf(type) === index);
  },

  // Получить все уникальные классы скорости
  getAllSpeedTypes: (): string[] => {
    const speedTypes = LOCAL_SHIPS_DATA.map(ship => ship.speedType);
    return speedTypes.filter((type, index) => speedTypes.indexOf(type) === index);
  }
};

// Маппинг типов кораблей на русские названия
export const SHIP_TYPE_NAMES: Record<string, string> = {
  'BB': 'Линейный корабль',
  'BC': 'Линейный крейсер',
  'CV': 'Авианосец',
  'CA': 'Тяжелый крейсер',
  'CL': 'Легкий крейсер',
  'DD': 'Эсминец',
  'CG': 'Сторожевой корабль',
  'TK': 'Танкер'
};

// Маппинг классов скорости на русские названия
export const SPEED_TYPE_NAMES: Record<string, string> = {
  'F': 'Быстрый',
  'M': 'Средний',
  'S': 'Медленный',
  'VS': 'Очень медленный'
};

// Маппинг классов скорости на максимальное расстояние движения
export const SPEED_TYPE_MAX_DISTANCE: Record<string, number> = {
  'F': 2,  // Быстрый - до 2 гексов
  'M': 1,  // Средний - 1 гекс
  'S': 1,  // Медленный - 1 гекс
  'VS': 1  // Очень медленный - 1 гекс
};

// Маппинг классов скорости на интервалы движения
export const SPEED_TYPE_MOVEMENT_INTERVAL: Record<string, number> = {
  'F': 1,  // Быстрый - может двигаться каждый ход
  'M': 1,  // Средний - может двигаться каждый ход
  'S': 2,  // Медленный - может двигаться через ход
  'VS': 2  // Очень медленный - может двигаться через ход
};

// Утилиты для работы с кораблями
export const shipUtils = {
  // Получить название типа корабля
  getShipTypeName: (type: string): string => {
    return SHIP_TYPE_NAMES[type] || type;
  },

  // Получить название класса скорости
  getSpeedTypeName: (speedType: string): string => {
    return SPEED_TYPE_NAMES[speedType] || speedType;
  },

  // Получить максимальное расстояние движения
  getMaxMovementDistance: (speedType: string): number => {
    return SPEED_TYPE_MAX_DISTANCE[speedType] || 1;
  },

  // Получить интервал движения
  getMovementInterval: (speedType: string): number => {
    return SPEED_TYPE_MOVEMENT_INTERVAL[speedType] || 1;
  },

  // Проверить, может ли корабль двигаться в этот ход
  canMoveThisTurn: (speedType: string, previousTurnMoved: number): boolean => {
    const interval = SPEED_TYPE_MOVEMENT_INTERVAL[speedType];
    return previousTurnMoved === 0 || interval === 1;
  },

  // Получить полное название корабля
  getFullShipName: (ship: ShipData): string => {
    const typeName = shipUtils.getShipTypeName(ship.type);
    const sideName = ship.side === 'german' ? 'Немецкий' : 'Британский';
    return `${sideName} ${typeName} ${ship.name}`;
  },

  // Получить описание корабля
  getShipDescription: (ship: ShipData): string => {
    const typeName = shipUtils.getShipTypeName(ship.type);
    const speedName = shipUtils.getSpeedTypeName(ship.speedType);
    return `${typeName}, класс скорости: ${speedName}, топливо: ${ship.maxFuel}`;
  },

  // Получить эффективный уровень радара (с учетом поломки)
  getEffectiveRadarLevel: (ship: ShipData, radarBroken?: boolean): number => {
    if (radarBroken) {
      return 0;
    }
    return ship.radarLevel;
  },

  // Получить описание радара
  getRadarDescription: (ship: ShipData, radarBroken?: boolean): string => {
    const effectiveLevel = shipUtils.getEffectiveRadarLevel(ship, radarBroken);
    if (effectiveLevel === 0) {
      return radarBroken ? 'Сломан' : 'Нет';
    }
    return `Уровень ${effectiveLevel}`;
  }
};
