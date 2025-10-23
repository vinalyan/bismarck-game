// Данные кораблей из игры "Погоня за Бисмарком"
import { ShipData } from '../types/shipTypes';

export const SHIPS_DATA: ShipData[] = [
  // НЕМЕЦКИЕ КОРАБЛИ
  {
    id: "bismarck",
    name: "BISMARCK",
    type: "BB",
    side: "german",
    maxFuel: 18,
    baseEvasion: 30,
    radarLevel: 0,
    hullBoxes: 12,
    basePrimaryArmamentBow: 8,
    basePrimaryArmamentStern: 2,
    baseSecondaryArmament: 8,
    maxTorpedos: 0,
    speedType: "M",
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
    id: "tirpitz",
    name: "TIRPITZ",
    type: "BB",
    side: "german",
    maxFuel: 19,
    baseEvasion: 30,
    radarLevel: 0,
    hullBoxes: 12,
    basePrimaryArmamentBow: 8,
    basePrimaryArmamentStern: 2,
    baseSecondaryArmament: 8,
    maxTorpedos: 4,
    speedType: "M",
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

  // СОЮЗНИЧЕСКИЕ КОРАБЛИ
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
    id: "sheffield",
    name: "SHEFFIELD",
    type: "CL",
    side: "allied",
    maxFuel: 8,
    baseEvasion: 32,
    radarLevel: 2,
    hullBoxes: 4,
    basePrimaryArmamentBow: 2,
    basePrimaryArmamentStern: 2,
    baseSecondaryArmament: 4,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "pow_hood_fl",
    name: "POW/HOOD FL.",
    type: "DD",
    side: "allied",
    maxFuel: 6,
    baseEvasion: 35,
    radarLevel: 1,
    hullBoxes: 8,
    basePrimaryArmamentBow: 6,
    basePrimaryArmamentStern: 6,
    baseSecondaryArmament: 0,
    maxTorpedos: 0,
    speedType: "F"
  },
  {
    id: "kgv_flotilla",
    name: "KGV FLOTILLA",
    type: "DD",
    side: "allied",
    maxFuel: 6,
    baseEvasion: 35,
    radarLevel: 1,
    hullBoxes: 8,
    basePrimaryArmamentBow: 5,
    basePrimaryArmamentStern: 5,
    baseSecondaryArmament: 0,
    maxTorpedos: 0,
    speedType: "F"
  }
];

// Утилиты для работы с данными кораблей
export const shipsUtils = {
  // Получить корабль по ID
  getShipById: (id: string): ShipData | undefined => {
    return SHIPS_DATA.find(ship => ship.id === id);
  },

  // Получить корабли по стороне
  getShipsBySide: (side: 'german' | 'allied'): ShipData[] => {
    return SHIPS_DATA.filter(ship => ship.side === side);
  },

  // Получить корабли по типу
  getShipsByType: (type: string): ShipData[] => {
    return SHIPS_DATA.filter(ship => ship.type === type);
  },

  // Получить корабли по классу скорости
  getShipsBySpeedType: (speedType: string): ShipData[] => {
    return SHIPS_DATA.filter(ship => ship.speedType === speedType);
  },

  // Получить все уникальные типы кораблей
  getAllShipTypes: (): string[] => {
    const types = SHIPS_DATA.map(ship => ship.type);
    return types.filter((type, index) => types.indexOf(type) === index);
  },

  // Получить все уникальные классы скорости
  getAllSpeedTypes: (): string[] => {
    const speedTypes = SHIPS_DATA.map(ship => ship.speedType);
    return speedTypes.filter((type, index) => speedTypes.indexOf(type) === index);
  }
};
