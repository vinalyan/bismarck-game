import axios from 'axios';
import { searchAPI } from './searchAPI';

// Мокируем axios
jest.mock('axios');
const mockedAxios = axios as jest.Mocked<typeof axios>;

describe('searchAPI', () => {
  const gameId = 'game-1';
  const token = 'test-token';

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getSearchFactors', () => {
    it('should call GET with correct params and return search factors', async () => {
      const hexIds = ['A1', 'A2', 'B1'];
      const playerSide = 'german' as const;

      const mockResponse = {
        data: {
          success: true,
          data: {
            hex_factors: {
              'A1': 5,
              'A2': 3,
              'B1': 2,
            },
            hex_markers: {
              'A1': { flight_path_search: 1 },
            },
          },
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await searchAPI.getSearchFactors(gameId, hexIds, playerSide, token);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/search/factors?hex_ids=A1,A2,B1&player_side=${playerSide}`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        }
      );
      expect(result.hex_factors).toEqual(mockResponse.data.data.hex_factors);
      expect(result.hex_markers).toEqual(mockResponse.data.data.hex_markers);
    });

    it('should handle empty response data', async () => {
      const hexIds = ['A1'];
      const playerSide = 'allied' as const;

      const mockResponse = {
        data: {
          success: true,
          data: {},
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await searchAPI.getSearchFactors(gameId, hexIds, playerSide, token);

      expect(result.hex_factors).toEqual({});
      expect(result.hex_markers).toEqual({});
    });

    it('should handle error response', async () => {
      const hexIds = ['A1'];
      const playerSide = 'german' as const;

      const errorResponse = {
        response: {
          data: {
            error: 'Failed to fetch search factors',
          },
        },
      };

      mockedAxios.get.mockRejectedValue(errorResponse);

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await searchAPI.getSearchFactors(gameId, hexIds, playerSide, token);

      expect(result.hex_factors).toEqual({});
      expect(result.hex_markers).toEqual({});
      expect(result.error).toBe('Failed to fetch search factors');
      expect(consoleErrorSpy).toHaveBeenCalled();

      consoleErrorSpy.mockRestore();
    });
  });

  describe('addHexMarker', () => {
    it('should call POST with correct data and return success', async () => {
      const hexId = 'A1';
      const markerType = 'flight_path_search';

      const mockResponse = {
        data: {
          success: true,
        },
      };

      mockedAxios.post.mockResolvedValue(mockResponse);

      const result = await searchAPI.addHexMarker(gameId, hexId, markerType, token);

      expect(mockedAxios.post).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/hex-markers`,
        { hex_id: hexId, marker_type: markerType },
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        }
      );
      expect(result.success).toBe(true);
    });

    it('should handle error response', async () => {
      const hexId = 'A1';
      const markerType = 'flight_path_search';

      const mockResponse = {
        data: {
          success: false,
          error: 'Failed to add marker',
        },
      };

      mockedAxios.post.mockResolvedValue(mockResponse);

      const result = await searchAPI.addHexMarker(gameId, hexId, markerType, token);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Failed to add marker');
      // console.error не вызывается при success: false, только при исключении
    });

    it('should handle network error', async () => {
      const hexId = 'A1';
      const markerType = 'flight_path_search';

      const errorResponse = {
        response: {
          data: {
            error: 'Network error',
          },
        },
      };

      mockedAxios.post.mockRejectedValue(errorResponse);

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await searchAPI.addHexMarker(gameId, hexId, markerType, token);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Network error');

      consoleErrorSpy.mockRestore();
    });
  });

  describe('removeHexMarker', () => {
    it('should call DELETE with correct params and return success', async () => {
      const hexId = 'A1';
      const markerType = 'flight_path_search';

      const mockResponse = {
        data: {
          success: true,
        },
      };

      mockedAxios.delete.mockResolvedValue(mockResponse);

      const result = await searchAPI.removeHexMarker(gameId, hexId, markerType, token);

      expect(mockedAxios.delete).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/hex-markers/${hexId}?type=${markerType}`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        }
      );
      expect(result.success).toBe(true);
    });

    it('should handle error response', async () => {
      const hexId = 'A1';
      const markerType = 'flight_path_search';

      const mockResponse = {
        data: {
          success: false,
          error: 'Failed to remove marker',
        },
      };

      mockedAxios.delete.mockResolvedValue(mockResponse);

      const result = await searchAPI.removeHexMarker(gameId, hexId, markerType, token);

      expect(result.success).toBe(false);
      expect(result.error).toBe('Failed to remove marker');
      // console.error не вызывается при success: false, только при исключении
    });
  });

  describe('getHexMarkers', () => {
    it('should call GET with correct params and return markers', async () => {
      const markerType = 'flight_path_search';

      const mockResponse = {
        data: {
          success: true,
          data: {
            markers: ['A1', 'A2', 'B1'],
          },
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const result = await searchAPI.getHexMarkers(gameId, markerType, token);

      expect(mockedAxios.get).toHaveBeenCalledWith(
        `http://localhost:8080/api/games/${gameId}/hex-markers?type=${markerType}`,
        {
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        }
      );
      expect(result).toEqual(['A1', 'A2', 'B1']);
    });

    it('should return empty array when response is invalid', async () => {
      const markerType = 'flight_path_search';

      const mockResponse = {
        data: {
          success: false,
        },
      };

      mockedAxios.get.mockResolvedValue(mockResponse);

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await searchAPI.getHexMarkers(gameId, markerType, token);

      expect(result).toEqual([]);

      consoleErrorSpy.mockRestore();
    });

    it('should handle error', async () => {
      const markerType = 'flight_path_search';

      mockedAxios.get.mockRejectedValue(new Error('Network error'));

      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

      const result = await searchAPI.getHexMarkers(gameId, markerType, token);

      expect(result).toEqual([]);

      consoleErrorSpy.mockRestore();
    });
  });
});

