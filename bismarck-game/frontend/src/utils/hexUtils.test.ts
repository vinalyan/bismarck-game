import {
  hex,
  hexAdd,
  hexSubtract,
  hexMultiply,
  hexLength,
  hexDistance,
  hexDirection,
  hexNeighbor,
  hexNeighbors,
  hexRound,
  hexToPixel,
  pixelToHex,
  hexCornerOffset,
  polygonCorners,
  offsetToCube,
  cubeToOffset,
  cubeDistance,
  offsetDistance,
  getCubeNeighbors,
  getClosestNeighbors,
  buildPath,
  createLayout,
  hexRange,
  hexRing,
  isValidHex,
  hexLerp,
  hexLineDraw,
  offsetToPixel,
  calculateMapSize,
  offsetPolygonCorners,
  LAYOUT_POINTY,
  LAYOUT_FLAT,
  MAP_CONSTANTS,
  Hex,
  FractionalHex,
  OffsetCoord,
  Layout,
  Point,
  Orientation
} from './hexUtils';

describe('hexUtils', () => {
  describe('hex()', () => {
    it('should create hex with correct q, r, s values', () => {
      const h = hex(1, 2, -3);
      expect(h.q).toBe(1);
      expect(h.r).toBe(2);
      expect(h.s).toBe(-3);
    });

    it('should auto-calculate s when not provided', () => {
      const h = hex(1, 2);
      expect(h.q).toBe(1);
      expect(h.r).toBe(2);
      expect(h.s).toBe(-3);
    });

    it('should ensure q + r + s = 0', () => {
      const h = hex(1, 2);
      expect(h.q + h.r + h.s).toBe(0);
    });

    it('should throw error for invalid hex coordinates', () => {
      expect(() => hex(1, 2, 3)).toThrow('Invalid hex coordinates');
    });

    it('should work with negative coordinates', () => {
      const h = hex(-1, -2);
      expect(h.q).toBe(-1);
      expect(h.r).toBe(-2);
      expect(h.s).toBe(3);
    });
  });

  describe('hexAdd()', () => {
    it('should add two hexes correctly', () => {
      const a = hex(1, 2, -3);
      const b = hex(2, 3, -5);
      const result = hexAdd(a, b);
      expect(result.q).toBe(3);
      expect(result.r).toBe(5);
      expect(result.s).toBe(-8);
      expect(result.q + result.r + result.s).toBe(0);
    });

    it('should handle negative coordinates', () => {
      const a = hex(1, 2);
      const b = hex(-2, -3);
      const result = hexAdd(a, b);
      expect(result.q).toBe(-1);
      expect(result.r).toBe(-1);
      expect(result.s).toBe(2);
    });
  });

  describe('hexSubtract()', () => {
    it('should subtract two hexes correctly', () => {
      const a = hex(3, 5);
      const b = hex(1, 2);
      const result = hexSubtract(a, b);
      expect(result.q).toBe(2);
      expect(result.r).toBe(3);
      expect(result.s).toBe(-5);
    });

    it('should handle negative results', () => {
      const a = hex(1, 2);
      const b = hex(3, 4);
      const result = hexSubtract(a, b);
      expect(result.q).toBe(-2);
      expect(result.r).toBe(-2);
      expect(result.s).toBe(4);
    });
  });

  describe('hexMultiply()', () => {
    it('should multiply hex by scalar correctly', () => {
      const h = hex(1, 2);
      const result = hexMultiply(h, 2);
      expect(result.q).toBe(2);
      expect(result.r).toBe(4);
      expect(result.s).toBe(-6);
    });

    it('should handle negative scalar', () => {
      const h = hex(1, 2);
      const result = hexMultiply(h, -2);
      expect(result.q).toBe(-2);
      expect(result.r).toBe(-4);
      expect(result.s).toBe(6);
    });

    it('should handle zero scalar', () => {
      const h = hex(1, 2);
      const result = hexMultiply(h, 0);
      expect(result.q).toBe(0);
      expect(result.r).toBe(0);
      // s can be 0 or -0, both are valid for hex coordinates
      expect(Math.abs(result.s)).toBe(0);
      expect(result.q + result.r + result.s).toBe(0);
    });
  });

  describe('hexLength()', () => {
    it('should calculate length correctly', () => {
      const h = hex(1, 2);
      const length = hexLength(h);
      expect(length).toBe((Math.abs(1) + Math.abs(2) + Math.abs(-3)) / 2);
      expect(length).toBe(3);
    });

    it('should return 0 for zero hex', () => {
      const h = hex(0, 0);
      expect(hexLength(h)).toBe(0);
    });

    it('should handle negative coordinates', () => {
      const h = hex(-2, -3);
      // For hex(-2, -3), s = -(-2) - (-3) = 2 + 3 = 5
      // Length = (|q| + |r| + |s|) / 2 = (2 + 3 + 5) / 2 = 5
      const length = hexLength(h);
      expect(length).toBe((Math.abs(-2) + Math.abs(-3) + Math.abs(5)) / 2);
      expect(length).toBe(5);
    });
  });

  describe('hexDistance()', () => {
    it('should calculate distance between adjacent hexes as 1', () => {
      const a = hex(0, 0);
      const b = hex(1, 0);
      expect(hexDistance(a, b)).toBe(1);
    });

    it('should calculate distance correctly for non-adjacent hexes', () => {
      const a = hex(0, 0);
      const b = hex(2, 3);
      const distance = hexDistance(a, b);
      expect(distance).toBeGreaterThan(1);
    });

    it('should return 0 for same hex', () => {
      const a = hex(1, 2);
      expect(hexDistance(a, a)).toBe(0);
    });

    it('should be symmetric', () => {
      const a = hex(1, 2);
      const b = hex(3, 4);
      expect(hexDistance(a, b)).toBe(hexDistance(b, a));
    });
  });

  describe('hexDirection()', () => {
    it('should return correct direction for valid index', () => {
      const dir = hexDirection(0);
      expect(dir).toBeDefined();
      expect(dir.q).toBe(1);
      expect(dir.r).toBe(0);
      expect(dir.s).toBe(-1);
    });

    it('should return all 6 directions', () => {
      for (let i = 0; i < 6; i++) {
        const dir = hexDirection(i);
        expect(dir).toBeDefined();
        expect(dir.q + dir.r + dir.s).toBe(0);
      }
    });
  });

  describe('hexNeighbor()', () => {
    it('should return correct neighbor', () => {
      const center = hex(0, 0);
      const neighbor = hexNeighbor(center, 0);
      expect(neighbor.q).toBe(1);
      expect(neighbor.r).toBe(0);
      expect(neighbor.s).toBe(-1);
    });

    it('should return different neighbors for different directions', () => {
      const center = hex(0, 0);
      const neighbors = [];
      for (let i = 0; i < 6; i++) {
        neighbors.push(hexNeighbor(center, i));
      }
      // Все соседи должны быть разными
      const unique = new Set(neighbors.map(n => `${n.q},${n.r},${n.s}`));
      expect(unique.size).toBe(6);
    });
  });

  describe('hexNeighbors()', () => {
    it('should return all 6 neighbors', () => {
      const center = hex(0, 0);
      const neighbors = hexNeighbors(center);
      expect(neighbors.length).toBe(6);
    });

    it('should return neighbors at distance 1', () => {
      const center = hex(0, 0);
      const neighbors = hexNeighbors(center);
      neighbors.forEach(neighbor => {
        expect(hexDistance(center, neighbor)).toBe(1);
      });
    });
  });

  describe('hexRound()', () => {
    it('should round fractional hex to nearest integer hex', () => {
      const fractional: FractionalHex = { q: 0.7, r: 0.3, s: -1.0 };
      const rounded = hexRound(fractional);
      expect(rounded.q + rounded.r + rounded.s).toBe(0);
      expect(hexDistance(rounded, hex(1, 0))).toBeLessThanOrEqual(1);
    });

    it('should handle exact integer values', () => {
      const fractional: FractionalHex = { q: 1, r: 2, s: -3 };
      const rounded = hexRound(fractional);
      expect(rounded.q).toBe(1);
      expect(rounded.r).toBe(2);
      expect(rounded.s).toBe(-3);
    });

    it('should maintain q + r + s = 0 constraint', () => {
      const fractional: FractionalHex = { q: 0.8, r: 1.2, s: -2.1 };
      const rounded = hexRound(fractional);
      expect(rounded.q + rounded.r + rounded.s).toBe(0);
    });
  });

  describe('hexToPixel()', () => {
    it('should convert hex to pixel coordinates', () => {
      const layout: Layout = {
        orientation: LAYOUT_POINTY,
        size: { x: 10, y: 10 },
        origin: { x: 0, y: 0 }
      };
      const h = hex(0, 0);
      const point = hexToPixel(layout, h);
      expect(point.x).toBeDefined();
      expect(point.y).toBeDefined();
    });

    it('should respect layout origin', () => {
      const layout: Layout = {
        orientation: LAYOUT_POINTY,
        size: { x: 10, y: 10 },
        origin: { x: 100, y: 100 }
      };
      const h = hex(0, 0);
      const point = hexToPixel(layout, h);
      expect(point.x).toBeCloseTo(100, 0);
      expect(point.y).toBeCloseTo(100, 0);
    });
  });

  describe('pixelToHex()', () => {
    it('should convert pixel to fractional hex', () => {
      const layout: Layout = {
        orientation: LAYOUT_POINTY,
        size: { x: 10, y: 10 },
        origin: { x: 0, y: 0 }
      };
      const point: Point = { x: 0, y: 0 };
      const fractional = pixelToHex(layout, point);
      expect(fractional.q + fractional.r + fractional.s).toBeCloseTo(0, 5);
    });

    it('should be approximately inverse of hexToPixel', () => {
      const layout: Layout = {
        orientation: LAYOUT_POINTY,
        size: { x: 10, y: 10 },
        origin: { x: 0, y: 0 }
      };
      const h = hex(1, 2);
      const point = hexToPixel(layout, h);
      const fractional = pixelToHex(layout, point);
      expect(fractional.q).toBeCloseTo(1, 1);
      expect(fractional.r).toBeCloseTo(2, 1);
    });
  });

  describe('offsetToCube()', () => {
    it('should convert offset coordinates to cube coordinates', () => {
      const offset: OffsetCoord = { col: 0, row: 0 };
      const cube = offsetToCube(offset);
      expect(cube.q + cube.r + cube.s).toBe(0);
    });

    it('should handle valid offset coordinates', () => {
      const offset: OffsetCoord = { col: 10, row: 5 };
      const cube = offsetToCube(offset);
      expect(cube.q + cube.r + cube.s).toBe(0);
    });
  });

  describe('cubeToOffset()', () => {
    it('should convert cube coordinates to offset coordinates', () => {
      const cube = hex(1, 2);
      const offset = cubeToOffset(cube);
      expect(offset.col).toBeDefined();
      expect(offset.row).toBeDefined();
    });

    it('should handle negative cube coordinates', () => {
      const cube = hex(-1, -2);
      const offset = cubeToOffset(cube);
      expect(offset.col).toBeDefined();
      expect(offset.row).toBeDefined();
    });
  });

  describe('cubeDistance()', () => {
    it('should calculate distance between cube coordinates', () => {
      const a = hex(0, 0);
      const b = hex(1, 0);
      expect(cubeDistance(a, b)).toBe(1);
    });

    it('should match hexDistance for same hexes', () => {
      const a = hex(1, 2);
      const b = hex(3, 4);
      expect(cubeDistance(a, b)).toBe(hexDistance(a, b));
    });

    it('should return 0 for same hex', () => {
      const a = hex(1, 2);
      expect(cubeDistance(a, a)).toBe(0);
    });
  });

  describe('offsetDistance()', () => {
    it('should calculate distance between offset coordinates', () => {
      const a: OffsetCoord = { col: 0, row: 0 };
      const b: OffsetCoord = { col: 1, row: 0 };
      const distance = offsetDistance(a, b);
      expect(distance).toBeGreaterThanOrEqual(0);
    });

    it('should return 0 for same coordinates', () => {
      const a: OffsetCoord = { col: 5, row: 5 };
      expect(offsetDistance(a, a)).toBe(0);
    });
  });

  describe('getCubeNeighbors()', () => {
    it('should return neighbors at distance 1 by default', () => {
      const center: OffsetCoord = { col: 10, row: 10 };
      const neighbors = getCubeNeighbors(center, 1);
      expect(neighbors.length).toBeGreaterThan(0);
      expect(neighbors.length).toBeLessThanOrEqual(6);
    });

    it('should return more neighbors for larger maxDistance', () => {
      const center: OffsetCoord = { col: 17, row: 17 }; // Center of map
      const neighbors1 = getCubeNeighbors(center, 1);
      const neighbors2 = getCubeNeighbors(center, 2);
      expect(neighbors2.length).toBeGreaterThan(neighbors1.length);
    });

    it('should filter neighbors within map bounds', () => {
      const center: OffsetCoord = { col: 0, row: 0 };
      const neighbors = getCubeNeighbors(center, 1);
      neighbors.forEach(neighbor => {
        expect(neighbor.col).toBeGreaterThanOrEqual(0);
        expect(neighbor.row).toBeGreaterThanOrEqual(0);
        expect(neighbor.col).toBeLessThan(MAP_CONSTANTS.HEX_GRID_WIDTH);
        expect(neighbor.row).toBeLessThan(MAP_CONSTANTS.HEX_GRID_HEIGHT);
      });
    });
  });

  describe('getClosestNeighbors()', () => {
    it('should return specified number of neighbors', () => {
      const center: OffsetCoord = { col: 17, row: 17 };
      const neighbors = getClosestNeighbors(center, 5);
      expect(neighbors.length).toBe(5);
    });

    it('should return neighbors sorted by distance', () => {
      const center: OffsetCoord = { col: 17, row: 17 };
      const neighbors = getClosestNeighbors(center, 5);
      for (let i = 1; i < neighbors.length; i++) {
        const dist1 = offsetDistance(center, neighbors[i - 1]);
        const dist2 = offsetDistance(center, neighbors[i]);
        expect(dist2).toBeGreaterThanOrEqual(dist1);
      }
    });
  });

  describe('buildPath()', () => {
    it('should build path between two hexes', () => {
      const from: OffsetCoord = { col: 0, row: 0 };
      const to: OffsetCoord = { col: 5, row: 5 };
      const path = buildPath(from, to);
      expect(path.length).toBeGreaterThan(1);
      expect(path[0]).toEqual(from);
      expect(path[path.length - 1]).toEqual(to);
    });

    it('should return both points for distance <= 1', () => {
      const from: OffsetCoord = { col: 5, row: 5 };
      const to: OffsetCoord = { col: 5, row: 6 };
      const path = buildPath(from, to);
      expect(path.length).toBe(2);
      expect(path[0]).toEqual(from);
      expect(path[1]).toEqual(to);
    });

    it('should return same point for distance 0', () => {
      const from: OffsetCoord = { col: 5, row: 5 };
      const to: OffsetCoord = { col: 5, row: 5 };
      const path = buildPath(from, to);
      expect(path.length).toBe(2);
      expect(path[0]).toEqual(from);
      expect(path[1]).toEqual(to);
    });
  });

  describe('hexRange()', () => {
    it('should return all hexes within range', () => {
      const center = hex(0, 0);
      const range = hexRange(center, 1);
      expect(range.length).toBe(7); // Center + 6 neighbors
    });

    it('should include center hex', () => {
      const center = hex(0, 0);
      const range = hexRange(center, 1);
      const hasCenter = range.some(h => h.q === 0 && h.r === 0 && h.s === 0);
      expect(hasCenter).toBe(true);
    });

    it('should return correct number for range 2', () => {
      const center = hex(0, 0);
      const range = hexRange(center, 2);
      // 1 (center) + 6 (distance 1) + 12 (distance 2) = 19
      expect(range.length).toBe(19);
    });
  });

  describe('hexRing()', () => {
    it('should return center for radius 0', () => {
      const center = hex(0, 0);
      const ring = hexRing(center, 0);
      expect(ring.length).toBe(1);
      expect(ring[0]).toEqual(center);
    });

    it('should return 6 hexes for radius 1', () => {
      const center = hex(0, 0);
      const ring = hexRing(center, 1);
      expect(ring.length).toBe(6);
      ring.forEach(h => {
        expect(hexDistance(center, h)).toBe(1);
      });
    });
  });

  describe('isValidHex()', () => {
    it('should return true for valid hex', () => {
      const h = hex(1, 2);
      expect(isValidHex(h)).toBe(true);
    });

    it('should return false for invalid hex', () => {
      const h: Hex = { q: 1, r: 2, s: 3 };
      expect(isValidHex(h)).toBe(false);
    });

    it('should handle rounding errors', () => {
      const h: Hex = { q: 1, r: 2, s: -3.0001 };
      // Should handle small rounding errors
      expect(isValidHex(h)).toBe(true);
    });
  });

  describe('hexLerp()', () => {
    it('should interpolate between two hexes', () => {
      const a: FractionalHex = { q: 0, r: 0, s: 0 };
      const b: FractionalHex = { q: 2, r: 2, s: -4 };
      const result = hexLerp(a, b, 0.5);
      expect(result.q).toBe(1);
      expect(result.r).toBe(1);
      expect(result.s).toBe(-2);
    });

    it('should return first hex for t=0', () => {
      const a: FractionalHex = { q: 1, r: 2, s: -3 };
      const b: FractionalHex = { q: 3, r: 4, s: -7 };
      const result = hexLerp(a, b, 0);
      expect(result.q).toBe(1);
      expect(result.r).toBe(2);
      expect(result.s).toBe(-3);
    });

    it('should return second hex for t=1', () => {
      const a: FractionalHex = { q: 1, r: 2, s: -3 };
      const b: FractionalHex = { q: 3, r: 4, s: -7 };
      const result = hexLerp(a, b, 1);
      expect(result.q).toBe(3);
      expect(result.r).toBe(4);
      expect(result.s).toBe(-7);
    });
  });

  describe('hexLineDraw()', () => {
    it('should draw line between two hexes', () => {
      const a = hex(0, 0);
      const b = hex(2, 2);
      const line = hexLineDraw(a, b);
      expect(line.length).toBeGreaterThan(1);
      expect(line[0]).toEqual(a);
      expect(line[line.length - 1]).toEqual(b);
    });

    it('should return single hex for same hexes', () => {
      const a = hex(1, 2);
      const line = hexLineDraw(a, a);
      expect(line.length).toBeGreaterThanOrEqual(1);
    });
  });

  describe('offsetToPixel()', () => {
    it('should convert offset coordinates to pixel coordinates', () => {
      const coord: OffsetCoord = { col: 0, row: 0 };
      const point = offsetToPixel(coord, MAP_CONSTANTS.DEFAULT_HEX_RADIUS);
      expect(point.x).toBeDefined();
      expect(point.y).toBeDefined();
    });

    it('should handle different hex radius (radius parameter exists but coordinates are based on fixed map size)', () => {
      const coord: OffsetCoord = { col: 10, row: 10 };
      const point1 = offsetToPixel(coord, 10);
      const point2 = offsetToPixel(coord, 20);
      // Note: offsetToPixel uses fixed map size, so coordinates don't depend on hexRadius parameter
      // The function accepts hexRadius but doesn't use it in calculations
      expect(point2.x).toBe(point1.x);
      expect(point2.y).toBe(point1.y);
    });

    it('should apply offset for odd rows', () => {
      const coordEven: OffsetCoord = { col: 10, row: 10 };
      const coordOdd: OffsetCoord = { col: 10, row: 11 };
      const pointEven = offsetToPixel(coordEven, MAP_CONSTANTS.DEFAULT_HEX_RADIUS);
      const pointOdd = offsetToPixel(coordOdd, MAP_CONSTANTS.DEFAULT_HEX_RADIUS);
      expect(pointOdd.x).toBeLessThan(pointEven.x);
    });
  });

  describe('calculateMapSize()', () => {
    it('should return fixed map size', () => {
      const size = calculateMapSize(100, 100, 10);
      expect(size.width).toBe(MAP_CONSTANTS.BACKGROUND_WIDTH);
      expect(size.height).toBe(MAP_CONSTANTS.BACKGROUND_HEIGHT);
    });
  });

  describe('offsetPolygonCorners()', () => {
    it('should return 6 corners for hex', () => {
      const coord: OffsetCoord = { col: 10, row: 10 };
      const corners = offsetPolygonCorners(coord, MAP_CONSTANTS.DEFAULT_HEX_RADIUS);
      expect(corners.length).toBe(6);
    });

    it('should return corners with valid coordinates', () => {
      const coord: OffsetCoord = { col: 10, row: 10 };
      const corners = offsetPolygonCorners(coord, MAP_CONSTANTS.DEFAULT_HEX_RADIUS);
      corners.forEach(corner => {
        expect(corner.x).toBeDefined();
        expect(corner.y).toBeDefined();
        expect(typeof corner.x).toBe('number');
        expect(typeof corner.y).toBe('number');
      });
    });
  });

  describe('polygonCorners()', () => {
    it('should return 6 corners for hex', () => {
      const layout: Layout = {
        orientation: LAYOUT_POINTY,
        size: { x: 10, y: 10 },
        origin: { x: 0, y: 0 }
      };
      const h = hex(0, 0);
      const corners = polygonCorners(layout, h);
      expect(corners.length).toBe(6);
    });
  });

  describe('createLayout()', () => {
    it('should create layout with given parameters', () => {
      const size: Point = { x: 10, y: 10 };
      const origin: Point = { x: 100, y: 100 };
      const layout = createLayout(LAYOUT_POINTY, size, origin);
      expect(layout.orientation).toBe(LAYOUT_POINTY);
      expect(layout.size).toEqual(size);
      expect(layout.origin).toEqual(origin);
    });
  });

  describe('LAYOUT_POINTY and LAYOUT_FLAT', () => {
    it('should have valid orientation constants', () => {
      expect(LAYOUT_POINTY).toBeDefined();
      expect(LAYOUT_FLAT).toBeDefined();
      expect(LAYOUT_POINTY.f0).toBeDefined();
      expect(LAYOUT_FLAT.f0).toBeDefined();
    });
  });

  describe('MAP_CONSTANTS', () => {
    it('should have valid map constants', () => {
      expect(MAP_CONSTANTS.HEX_GRID_WIDTH).toBeGreaterThan(0);
      expect(MAP_CONSTANTS.HEX_GRID_HEIGHT).toBeGreaterThan(0);
      expect(MAP_CONSTANTS.DEFAULT_HEX_RADIUS).toBeGreaterThan(0);
    });
  });
});

