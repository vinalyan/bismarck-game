import {
  shipStateUtils,
  ShipGameState
} from './shipStateUtils';
import { ShipData } from '../types/shipTypes';

describe('shipStateUtils', () => {
  const mockShip: ShipData = {
    id: 'test-ship',
    name: 'Test Ship',
    type: 'BB',
    side: 'german',
    maxFuel: 100,
    baseEvasion: 30,
    radarLevel: 2,
    hullBoxes: 10,
    basePrimaryArmamentBow: 8,
    basePrimaryArmamentStern: 8,
    baseSecondaryArmament: 4,
    maxTorpedos: 0,
    speedType: 'M',
    specialRules: []
  };

  const mockBismarckShip: ShipData = {
    id: 'bismarck',
    name: 'Bismarck',
    type: 'BB',
    side: 'german',
    maxFuel: 100,
    baseEvasion: 30,
    radarLevel: 2,
    hullBoxes: 10,
    basePrimaryArmamentBow: 8,
    basePrimaryArmamentStern: 8,
    baseSecondaryArmament: 4,
    maxTorpedos: 0,
    speedType: 'M',
    specialRules: [
      {
        type: 'radar_loss_after_first_round',
        isActive: true,
        description: 'Radar breaks after first combat round'
      }
    ]
  };

  describe('createInitialState()', () => {
    it('should create initial state with ship id', () => {
      const state = shipStateUtils.createInitialState(mockShip);
      expect(state.shipId).toBe('test-ship');
    });

    it('should set currentFuel to maxFuel', () => {
      const state = shipStateUtils.createInitialState(mockShip);
      expect(state.currentFuel).toBe(mockShip.maxFuel);
    });

    it('should set radarBroken to false', () => {
      const state = shipStateUtils.createInitialState(mockShip);
      expect(state.radarBroken).toBe(false);
    });

    it('should set combatRoundsParticipated to 0', () => {
      const state = shipStateUtils.createInitialState(mockShip);
      expect(state.combatRoundsParticipated).toBe(0);
    });

    it('should set isDamaged to false', () => {
      const state = shipStateUtils.createInitialState(mockShip);
      expect(state.isDamaged).toBe(false);
    });

    it('should set damageLevel to 0', () => {
      const state = shipStateUtils.createInitialState(mockShip);
      expect(state.damageLevel).toBe(0);
    });
  });

  describe('shouldBreakRadarAfterCombat()', () => {
    it('should return false for regular ship', () => {
      const state = shipStateUtils.createInitialState(mockShip);
      const shouldBreak = shipStateUtils.shouldBreakRadarAfterCombat(mockShip, state);
      expect(shouldBreak).toBe(false);
    });

    it('should return false for Bismarck before first combat round', () => {
      const state = shipStateUtils.createInitialState(mockBismarckShip);
      const shouldBreak = shipStateUtils.shouldBreakRadarAfterCombat(mockBismarckShip, state);
      expect(shouldBreak).toBe(false);
    });

    it('should return true for Bismarck after first combat round', () => {
      const state: ShipGameState = {
        ...shipStateUtils.createInitialState(mockBismarckShip),
        combatRoundsParticipated: 1
      };
      const shouldBreak = shipStateUtils.shouldBreakRadarAfterCombat(mockBismarckShip, state);
      expect(shouldBreak).toBe(true);
    });

    it('should return false if special rule is not active', () => {
      const inactiveBismarck: ShipData = {
        ...mockBismarckShip,
        specialRules: [
          {
            type: 'radar_loss_after_first_round',
            isActive: false,
            description: 'Inactive rule'
          }
        ]
      };
      const state: ShipGameState = {
        ...shipStateUtils.createInitialState(inactiveBismarck),
        combatRoundsParticipated: 1
      };
      const shouldBreak = shipStateUtils.shouldBreakRadarAfterCombat(inactiveBismarck, state);
      expect(shouldBreak).toBe(false);
    });
  });

  describe('updateAfterCombatRound()', () => {
    it('should increment combatRoundsParticipated', () => {
      const initialState = shipStateUtils.createInitialState(mockShip);
      const newState = shipStateUtils.updateAfterCombatRound(mockShip, initialState);
      expect(newState.combatRoundsParticipated).toBe(1);
    });

    it('should not break radar for regular ship', () => {
      const initialState = shipStateUtils.createInitialState(mockShip);
      const newState = shipStateUtils.updateAfterCombatRound(mockShip, initialState);
      expect(newState.radarBroken).toBe(false);
    });

    it('should break radar for Bismarck after first round', () => {
      const initialState = shipStateUtils.createInitialState(mockBismarckShip);
      const newState = shipStateUtils.updateAfterCombatRound(mockBismarckShip, initialState);
      expect(newState.combatRoundsParticipated).toBe(1);
      expect(newState.radarBroken).toBe(true);
    });

    it('should preserve other state properties', () => {
      const initialState = shipStateUtils.createInitialState(mockShip);
      initialState.currentFuel = 50;
      const newState = shipStateUtils.updateAfterCombatRound(mockShip, initialState);
      expect(newState.currentFuel).toBe(50);
      expect(newState.shipId).toBe(initialState.shipId);
    });
  });

  describe('updateFuelAfterMovement()', () => {
    it('should decrease fuel by fuelCost', () => {
      const initialState = shipStateUtils.createInitialState(mockShip);
      const newState = shipStateUtils.updateFuelAfterMovement(mockShip, initialState, 10);
      expect(newState.currentFuel).toBe(90);
    });

    it('should not allow negative fuel', () => {
      const initialState = shipStateUtils.createInitialState(mockShip);
      initialState.currentFuel = 5;
      const newState = shipStateUtils.updateFuelAfterMovement(mockShip, initialState, 10);
      expect(newState.currentFuel).toBe(0);
    });

    it('should handle zero fuel cost', () => {
      const initialState = shipStateUtils.createInitialState(mockShip);
      initialState.currentFuel = 50;
      const newState = shipStateUtils.updateFuelAfterMovement(mockShip, initialState, 0);
      expect(newState.currentFuel).toBe(50);
    });

    it('should preserve other state properties', () => {
      const initialState = shipStateUtils.createInitialState(mockShip);
      initialState.combatRoundsParticipated = 2;
      const newState = shipStateUtils.updateFuelAfterMovement(mockShip, initialState, 10);
      expect(newState.combatRoundsParticipated).toBe(2);
      expect(newState.shipId).toBe(initialState.shipId);
    });
  });

  describe('getEffectiveRadarLevel()', () => {
    it('should return ship radar level when radar is not broken', () => {
      const state = shipStateUtils.createInitialState(mockShip);
      const level = shipStateUtils.getEffectiveRadarLevel(mockShip, state);
      expect(level).toBe(mockShip.radarLevel);
    });

    it('should return 0 when radar is broken', () => {
      const state: ShipGameState = {
        ...shipStateUtils.createInitialState(mockShip),
        radarBroken: true
      };
      const level = shipStateUtils.getEffectiveRadarLevel(mockShip, state);
      expect(level).toBe(0);
    });

    it('should return correct level for different radar levels', () => {
      const shipWithLevel3: ShipData = {
        ...mockShip,
        radarLevel: 3
      };
      const state = shipStateUtils.createInitialState(shipWithLevel3);
      const level = shipStateUtils.getEffectiveRadarLevel(shipWithLevel3, state);
      expect(level).toBe(3);
    });
  });

  describe('getRadarDescription()', () => {
    it('should return level description when radar is not broken', () => {
      const state = shipStateUtils.createInitialState(mockShip);
      const description = shipStateUtils.getRadarDescription(mockShip, state);
      expect(description).toBe('Уровень 2');
    });

    it('should return "Сломан" when radar is broken', () => {
      const state: ShipGameState = {
        ...shipStateUtils.createInitialState(mockShip),
        radarBroken: true
      };
      const description = shipStateUtils.getRadarDescription(mockShip, state);
      expect(description).toBe('Сломан');
    });

    it('should return "Нет" when radar level is 0 and not broken', () => {
      const shipNoRadar: ShipData = {
        ...mockShip,
        radarLevel: 0
      };
      const state = shipStateUtils.createInitialState(shipNoRadar);
      const description = shipStateUtils.getRadarDescription(shipNoRadar, state);
      expect(description).toBe('Нет');
    });
  });

  describe('updateAfterMovement()', () => {
    it('should update currentFuel', () => {
      const initialState = shipStateUtils.createInitialState(mockShip);
      const newState = shipStateUtils.updateAfterMovement(mockShip, initialState, 1, 80);
      expect(newState.currentFuel).toBe(80);
    });

    it('should update lastMovementTurn', () => {
      const initialState = shipStateUtils.createInitialState(mockShip);
      const turn = 5;
      const newState = shipStateUtils.updateAfterMovement(mockShip, initialState, turn, 90);
      expect(newState.lastMovementTurn).toBe(turn);
    });

    it('should preserve other state properties', () => {
      const initialState = shipStateUtils.createInitialState(mockShip);
      initialState.combatRoundsParticipated = 2;
      initialState.radarBroken = false;
      const newState = shipStateUtils.updateAfterMovement(mockShip, initialState, 1, 80);
      expect(newState.combatRoundsParticipated).toBe(2);
      expect(newState.radarBroken).toBe(false);
      expect(newState.shipId).toBe(initialState.shipId);
    });
  });
});

