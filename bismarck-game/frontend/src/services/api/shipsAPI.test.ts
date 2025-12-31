import axios from 'axios';
import { shipsAPI, ShipConfig } from './shipsAPI';

// Мокируем axios
jest.mock('axios');
const mockedAxios = axios as jest.Mocked<typeof axios>;

describe('shipsAPI', () => {
  const mockShipConfig: ShipConfig = {
    id: 'ship-1',
    name: 'Bismarck',
    type: 'battleship',
    side: 'german',
    maxFuel: 100,
    baseEvasion: 5,
    radarLevel: 3,
    hullBoxes: 10,
    basePrimaryArmamentBow: 8,
    basePrimaryArmamentStern: 8,
    baseSecondaryArmament: 4,
    maxTorpedos: 0,
    speedType: 'M',
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getAllShips', () => {
    it('should call GET /api/ships/all and return ships', async () => {
      const mockResponse = {
        data: {
          success: true,
          data: [mockShipConfig],
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await shipsAPI.getAllShips();

      expect(mockedAxios.get).toHaveBeenCalledWith('http://localhost:8080/api/ships/all');
      expect(result).toEqual([mockShipConfig]);
    });

    it('should return empty array on error', async () => {
      mockedAxios.get.mockRejectedValue(new Error('Network error'));

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await shipsAPI.getAllShips();

      expect(result).toEqual([]);
      expect(consoleErrorSpy).toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });

    it('should return empty array when success is false', async () => {
      const mockResponse = {
        data: {
          success: false,
          error: 'Failed to fetch ships',
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await shipsAPI.getAllShips();

      expect(result).toEqual([]);

      consoleErrorSpy.mockRestore();
    });
  });

  describe('getShipsBySide', () => {
    it('should call GET /api/ships/side/{side} and return ships', async () => {
      const side = 'german';

      const mockResponse = {
        data: {
          success: true,
          data: [mockShipConfig],
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await shipsAPI.getShipsBySide(side);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/ships/side/${side}`
      );
      expect(result).toEqual([mockShipConfig]);
    });

    it('should return empty array on error', async () => {
      mockedAxios.get.mockRejectedValue(new Error('Network error'));

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await shipsAPI.getShipsBySide('german');

      expect(result).toEqual([]);

      consoleErrorSpy.mockRestore();
    });
  });

  describe('getShipsByType', () => {
    it('should call GET /api/ships/type/{type} and return ships', async () => {
      const type = 'battleship';

      const mockResponse = {
        data: {
          success: true,
          data: [mockShipConfig],
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await shipsAPI.getShipsByType(type);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/ships/type/${type}`
      );
      expect(result).toEqual([mockShipConfig]);
    });

    it('should return empty array on error', async () => {
      mockedAxios.get.mockRejectedValue(new Error('Network error'));

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await shipsAPI.getShipsByType('battleship');

      expect(result).toEqual([]);

      consoleErrorSpy.mockRestore();
    });
  });

  describe('getShipConfig', () => {
    it('should call GET /api/ships/config/{shipId} and return ship config', async () => {
      const shipId = 'ship-1';

      const mockResponse = {
        data: {
          success: true,
          data: mockShipConfig,
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await shipsAPI.getShipConfig(shipId);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/ships/config/${shipId}`
      );
      expect(result).toEqual(mockShipConfig);
    });

    it('should return null on error', async () => {
      mockedAxios.get.mockRejectedValue(new Error('Network error'));

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await shipsAPI.getShipConfig('ship-1');

      expect(result).toBeNull();

      consoleErrorSpy.mockRestore();
    });

    it('should return null when success is false', async () => {
      const mockResponse = {
        data: {
          success: false,
          error: 'Ship not found',
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await shipsAPI.getShipConfig('ship-1');

      expect(result).toBeNull();

      consoleErrorSpy.mockRestore();
    });
  });

  describe('getShipConfigByTypeAndSide', () => {
    it('should call getShipsByType and find ship by side', async () => {
      const type = 'battleship';
      const side = 'german';

      const mockResponse = {
        data: {
          success: true,
          data: [mockShipConfig],
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await shipsAPI.getShipConfigByTypeAndSide(type, side);

      expect(result).toEqual(mockShipConfig);
    });

    it('should return null when ship not found', async () => {
      const type = 'battleship';
      const side = 'allied';

      const mockResponse = {
        data: {
          success: true,
          data: [mockShipConfig], // Only german ship
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await shipsAPI.getShipConfigByTypeAndSide(type, side);

      expect(result).toBeNull();
    });

    it('should return null on error', async () => {
      mockedAxios.get.mockRejectedValue(new Error('Network error'));

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await shipsAPI.getShipConfigByTypeAndSide('battleship', 'german');

      expect(result).toBeNull();

      consoleErrorSpy.mockRestore();
    });
  });

  describe('getConfigStats', () => {
    it('should call GET /api/ships/stats and return stats', async () => {
      const mockStats = {
        total_ships: 50,
        by_side: { german: 25, allied: 25 },
        by_type: { battleship: 5, cruiser: 20 },
      };

      const mockResponse = {
        data: {
          success: true,
          data: mockStats,
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await shipsAPI.getConfigStats();

      expect(mockedAxios.get).toHaveBeenCalledWith('http://localhost:8080/api/ships/stats');
      expect(result).toEqual(mockStats);
    });

    it('should return null on error', async () => {
      mockedAxios.get.mockRejectedValue(new Error('Network error'));

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await shipsAPI.getConfigStats();

      expect(result).toBeNull();

      consoleErrorSpy.mockRestore();
    });
  });
});

