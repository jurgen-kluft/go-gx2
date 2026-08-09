(function attachPaletteToolUtils(globalScope) {
  const DEFAULT_MAX_PALETTE_SIZE = 65535;

  function colorKey(r, g, b) {
    return (r << 16) | (g << 8) | b;
  }

  function clamp(value, min, max) {
    return Math.min(max, Math.max(min, value));
  }

  function paletteColorToCss(color) {
    return `rgb(${color.r}, ${color.g}, ${color.b})`;
  }

  function nearestColorIndex(r, g, b, palette) {
    if (!palette.length) {
      return -1;
    }

    let bestIndex = 0;
    let bestDistance = Number.POSITIVE_INFINITY;

    for (let i = 0; i < palette.length; i += 1) {
      const candidate = palette[i];
      const dr = r - candidate.r;
      const dg = g - candidate.g;
      const db = b - candidate.b;
      const distance = (dr * dr) + (dg * dg) + (db * db);
      if (distance < bestDistance) {
        bestDistance = distance;
        bestIndex = i;
      }
    }

    return bestIndex;
  }

  function extractPaletteAndIndexFromImageData(imageData, options = {}) {
    if (!imageData) {
      throw new Error('ImageData is required to extract palette and indices.');
    }

    const maxPaletteSize = clamp(
      Number(options.maxPaletteSize) || DEFAULT_MAX_PALETTE_SIZE,
      1,
      DEFAULT_MAX_PALETTE_SIZE,
    );

    const pixelCount = imageData.width * imageData.height;
    const palette = [];
    const indexImage = new Uint16Array(pixelCount);
    const colorToIndex = new Map();
    const overflowColorCache = new Map();

    const data = imageData.data;
    let overflowMappedCount = 0;

    for (let i = 0, pixel = 0; i < data.length; i += 4, pixel += 1) {
      const r = data[i];
      const g = data[i + 1];
      const b = data[i + 2];
      const key = colorKey(r, g, b);

      const existing = colorToIndex.get(key);
      if (existing !== undefined) {
        indexImage[pixel] = existing;
        palette[existing].count += 1;
        continue;
      }

      if (palette.length < maxPaletteSize) {
        const nextIndex = palette.length;
        palette.push({ r, g, b, count: 1 });
        colorToIndex.set(key, nextIndex);
        indexImage[pixel] = nextIndex;
        continue;
      }

      let nearest = overflowColorCache.get(key);
      if (nearest === undefined) {
        nearest = nearestColorIndex(r, g, b, palette);
        overflowColorCache.set(key, nearest);
      }

      indexImage[pixel] = nearest;
      palette[nearest].count += 1;
      overflowMappedCount += 1;
    }

    return {
      palette,
      indexImage,
      uniqueColorCount: colorToIndex.size + overflowColorCache.size,
      overflowMappedCount,
    };
  }

  function reindexToPalette(imageData, palette) {
    if (!imageData) {
      throw new Error('ImageData is required to reindex.');
    }
    if (!palette || !palette.length) {
      throw new Error('Palette cannot be empty when reindexing.');
    }

    const pixelCount = imageData.width * imageData.height;
    const indexImage = new Uint16Array(pixelCount);
    const colorCounts = new Uint32Array(palette.length);
    const keyCache = new Map();
    const data = imageData.data;

    for (let i = 0, pixel = 0; i < data.length; i += 4, pixel += 1) {
      const r = data[i];
      const g = data[i + 1];
      const b = data[i + 2];
      const key = colorKey(r, g, b);

      let index = keyCache.get(key);
      if (index === undefined) {
        index = nearestColorIndex(r, g, b, palette);
        keyCache.set(key, index);
      }

      indexImage[pixel] = index;
      colorCounts[index] += 1;
    }

    for (let i = 0; i < palette.length; i += 1) {
      palette[i].count = colorCounts[i];
    }

    return indexImage;
  }

  globalScope.PaletteToolUtils = {
    clamp,
    paletteColorToCss,
    nearestColorIndex,
    extractPaletteAndIndexFromImageData,
    reindexToPalette,
  };
})(window);
