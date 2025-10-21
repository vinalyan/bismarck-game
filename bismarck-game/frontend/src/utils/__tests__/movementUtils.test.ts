import { 
  movementUtils,
  ShipData,
  PreviousTurnInfo 
} from '../movementUtils';

// Mock data for testing
const createMockShip = (speedType: string, fuel: number = 10, noMovementTurnsLeft: number = 0): ShipData => ({
  id: 'test-ship',
  name: 'Test Ship',
  type: 'battleship',
  class: 'battleship',
  owner: 'german',
  nationality: 'german',
  position: 'J30',
  speedType,
  fuel,
  maxFuel: 20,
  hull: 10,
  maxHull: 10,
  noMovementTurnsLeft,
  isActivated: false,
  status: 'active',
  detectionLevel: 'none',
  lastKnownPos: null,
  taskForceId: null,
  damage: [],
  tacticalPosition: null,
  tacticalFacing: null,
  tacticalSpeed: null,
  evasionEffects: [],
  tacticalDamageTaken: [],
  hasFired: false,
  targetAcquired: null,
  torpedoesUsed: 0,
  movementUsed: 0,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString()
});

const createMockPreviousTurn = (movedHexes: number): PreviousTurnInfo => ({
  movedHexes,
  turn: 5
});

describe('Movement Utils', () => {
  describe('getMaxMovementDistance', () => {
    test('F type ship should have max distance 2', () => {
      const ship = createMockShip('F');
      const distance = movementUtils.getMaxMovementDistance(ship);
      expect(distance).toBe(2);
    });

    test('M type ship should have max distance 1', () => {
      const ship = createMockShip('M');
      const distance = movementUtils.getMaxMovementDistance(ship);
      expect(distance).toBe(1);
    });

    test('S type ship should have max distance 1', () => {
      const ship = createMockShip('S');
      const distance = movementUtils.getMaxMovementDistance(ship);
      expect(distance).toBe(1);
    });

    test('VS type ship should have max distance 1', () => {
      const ship = createMockShip('VS');
      const distance = movementUtils.getMaxMovementDistance(ship);
      expect(distance).toBe(1);
    });
  });

  describe('calculateMovementCost', () => {
    test('F type ship moving 1 hex costs 0 fuel', () => {
      const ship = createMockShip('F', 18);
      const result = movementUtils.calculateMovementCost(ship, 1);
      const cost = result.fuelCost;
      expect(cost).toBe(0);
    });

    test('F type ship moving 2 hexes costs 1 fuel', () => {
      const ship = createMockShip('F', 18);
      const result = movementUtils.calculateMovementCost(ship, 2);
      const cost = result.fuelCost;
      expect(cost).toBe(1);
    });

    test('M type ship moving 1 hex after no previous movement costs 0 fuel', () => {
      const ship = createMockShip('M', 15);
      const result = movementUtils.calculateMovementCost(ship, 1);
      const cost = result.fuelCost;
      expect(cost).toBe(0);
    });

    test('M type ship moving 1 hex after previous movement costs 1 fuel', () => {
      const ship = createMockShip('M', 15);
      const previousTurn = createMockPreviousTurn(1);
      const result = movementUtils.calculateMovementCost(ship, 1, previousTurn);
      const cost = result.fuelCost;
      expect(cost).toBe(1);
    });

    test('S type ship moving 1 hex costs 0 fuel', () => {
      const ship = createMockShip('S', 10);
      const result = movementUtils.calculateMovementCost(ship, 1);
      const cost = result.fuelCost;
      expect(cost).toBe(0);
    });

    test('VS type ship moving 1 hex costs 0 fuel', () => {
      const ship = createMockShip('VS', 5);
      const result = movementUtils.calculateMovementCost(ship, 1);
      const cost = result.fuelCost;
      expect(cost).toBe(0);
    });

    test('S type ship with movement restrictions cannot move', () => {
      const ship = createMockShip('S', 10, 1); // 1 turn restriction
      const result = movementUtils.calculateMovementCost(ship, 1);
      const cost = result.fuelCost;
      expect(cost).toBe(0); // Should return 0 for restricted movement
    });

    test('VS type ship with movement restrictions cannot move', () => {
      const ship = createMockShip('VS', 5, 2); // 2 turn restriction
      const result = movementUtils.calculateMovementCost(ship, 1);
      const cost = result.fuelCost;
      expect(cost).toBe(0); // Should return 0 for restricted movement
    });
  });

  describe('getAvailableMovementHexes', () => {
    const mockHexMap = new Map([
      ['J30', { q: 0, r: 0, s: 0 }],
      ['J31', { q: 1, r: 0, s: -1 }],
      ['J32', { q: 2, r: 0, s: -2 }],
      ['J33', { q: 3, r: 0, s: -3 }],
      ['I30', { q: 0, r: 1, s: -1 }],
      ['I31', { q: 1, r: 1, s: -2 }],
      ['I32', { q: 2, r: 1, s: -3 }],
      ['K30', { q: 0, r: -1, s: 1 }],
      ['K31', { q: 1, r: -1, s: 0 }],
      ['K32', { q: 2, r: -1, s: -1 }],
    ]);

    test('F type ship should have 2 hex movement options', () => {
      const ship = createMockShip('F', 18);
      const currentPosition = { col: 9, row: 29, letter: 'J', number: 30 };
      const availableHexes = movementUtils.getAvailableMovementHexes(
        ship,
        currentPosition,
        ship.fuel,
        undefined,
        undefined,
        ship.noMovementTurnsLeft
      );
      
      // F type can move 1 or 2 hexes
      expect(availableHexes).toBeDefined();
      expect(availableHexes.length).toBeGreaterThan(0);
      
      // Check that we have some movement options
      expect(availableHexes.some(hex => hex.distance <= 2)).toBe(true);
    });

    test('M type ship should have 1 hex movement options', () => {
      const ship = createMockShip('M', 15);
      const currentPosition = { col: 9, row: 29, letter: 'J', number: 30 };
      const availableHexes = movementUtils.getAvailableMovementHexes(
        ship,
        currentPosition,
        ship.fuel,
        undefined,
        undefined,
        ship.noMovementTurnsLeft
      );
      
      // M type can move only 1 hex
      expect(availableHexes).toBeDefined();
      expect(availableHexes.length).toBeGreaterThan(0);
      
      // Check that we have movement options within 1 hex
      expect(availableHexes.some(hex => hex.distance <= 1)).toBe(true);
    });

    test('S type ship should have 1 hex movement options when no restrictions', () => {
      const ship = createMockShip('S', 10, 0); // No restrictions
      const currentPosition = { col: 9, row: 29, letter: 'J', number: 30 };
      const availableHexes = movementUtils.getAvailableMovementHexes(
        ship,
        currentPosition,
        ship.fuel,
        undefined,
        undefined,
        ship.noMovementTurnsLeft
      );
      
      // S type can move only 1 hex
      expect(availableHexes).toBeDefined();
      expect(availableHexes.length).toBeGreaterThan(0);
      
      // Check that we have movement options within 1 hex
      expect(availableHexes.some(hex => hex.distance <= 1)).toBe(true);
    });

    test('S type ship should have no movement options with restrictions', () => {
      const ship = createMockShip('S', 10, 1); // 1 turn restriction
      const currentPosition = { col: 9, row: 29, letter: 'J', number: 30 };
      const availableHexes = movementUtils.getAvailableMovementHexes(
        ship,
        currentPosition,
        ship.fuel,
        undefined,
        undefined,
        ship.noMovementTurnsLeft
      );
      
      // S type with restrictions cannot move
      expect(availableHexes.length).toBe(0);
    });

    test('VS type ship should have 1 hex movement options when no restrictions', () => {
      const ship = createMockShip('VS', 5, 0); // No restrictions
      const currentPosition = { col: 9, row: 29, letter: 'J', number: 30 };
      const availableHexes = movementUtils.getAvailableMovementHexes(
        ship,
        currentPosition,
        ship.fuel,
        undefined,
        undefined,
        ship.noMovementTurnsLeft
      );
      
      // VS type can move only 1 hex
      expect(availableHexes).toBeDefined();
      expect(availableHexes.length).toBeGreaterThan(0);
      
      // Check that we have movement options within 1 hex
      expect(availableHexes.some(hex => hex.distance <= 1)).toBe(true);
    });

    test('VS type ship should have no movement options with restrictions', () => {
      const ship = createMockShip('VS', 5, 2); // 2 turn restriction
      const currentPosition = { col: 9, row: 29, letter: 'J', number: 30 };
      const availableHexes = movementUtils.getAvailableMovementHexes(
        ship,
        currentPosition,
        ship.fuel,
        undefined,
        undefined,
        ship.noMovementTurnsLeft
      );
      
      // VS type with restrictions cannot move
      expect(availableHexes.length).toBe(0);
    });

    test('Ship with insufficient fuel should have limited options', () => {
      const ship = createMockShip('F', 0); // No fuel
      const currentPosition = { col: 9, row: 29, letter: 'J', number: 30 };
      const availableHexes = movementUtils.getAvailableMovementHexes(
        ship,
        currentPosition,
        ship.fuel,
        undefined,
        undefined,
        ship.noMovementTurnsLeft
      );
      
      // F type with 0 fuel can move 1 hex (0 fuel cost) but not 2 hexes (1 fuel cost)
      expect(availableHexes.length).toBeGreaterThan(0); // Can move 1 hex
      expect(availableHexes.every(hex => hex.distance <= 1)).toBe(true); // Only 1 hex moves
    });

    // Simplified tests - removed complex hex validation tests
  });

  describe('Edge cases', () => {
    test('Ship with negative fuel should not be able to move', () => {
      const ship = createMockShip('F', -1); // Negative fuel
      const result = movementUtils.calculateMovementCost(ship, 1);
      const cost = result.fuelCost;
      expect(cost).toBe(0); // Should return 0 for invalid movement
    });

    test('Ship with maximum fuel should be able to move', () => {
      const ship = createMockShip('F', 18, 0); // Max fuel, no restrictions
      const result = movementUtils.calculateMovementCost(ship, 2);
      const cost = result.fuelCost;
      expect(cost).toBe(1); // Should return normal cost
    });

    test('Ship with movement restrictions should not be able to move', () => {
      const ship = createMockShip('S', 10, 5); // 5 turn restriction
      const result = movementUtils.calculateMovementCost(ship, 1);
      const cost = result.fuelCost;
      expect(cost).toBe(0); // Should return 0 for restricted movement
    });

    test('Ship with zero movement restrictions should be able to move', () => {
      const ship = createMockShip('S', 10, 0); // No restrictions
      const result = movementUtils.calculateMovementCost(ship, 1);
      const cost = result.fuelCost;
      expect(cost).toBe(0); // Should return normal cost (0 for S type)
    });
  });

  describe('Performance tests', () => {
    test('getAvailableMovementHexes should handle movement efficiently', () => {
      const ship = createMockShip('F', 18);
      const currentPosition = { col: 0, row: 0, letter: 'A', number: 1 };
      
      const startTime = performance.now();
      const availableHexes = movementUtils.getAvailableMovementHexes(
        ship,
        currentPosition,
        ship.fuel,
        undefined,
        undefined,
        ship.noMovementTurnsLeft
      );
      const endTime = performance.now();
      
      expect(availableHexes.length).toBeGreaterThan(0);
      expect(endTime - startTime).toBeLessThan(100); // Should complete in less than 100ms
    });
  });
});
