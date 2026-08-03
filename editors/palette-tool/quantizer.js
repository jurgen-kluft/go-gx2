(function attachPaletteToolQuantizer(globalScope) {
  const MAX_PNN_INPUT_COLORS = 2048;

  function colorKey(r, g, b) {
    return (r << 16) | (g << 8) | b;
  }

  function clamp(value, min, max) {
    return Math.min(max, Math.max(min, value));
  }

  function buildHistogram(imageData) {
    const histogram = new Map();
    const data = imageData.data;

    for (let i = 0; i < data.length; i += 4) {
      const r = data[i];
      const g = data[i + 1];
      const b = data[i + 2];
      const key = colorKey(r, g, b);
      histogram.set(key, (histogram.get(key) || 0) + 1);
    }

    return histogram;
  }

  function histogramToClusters(histogram) {
    const clusters = [];
    let nextId = 0;
    histogram.forEach((count, key) => {
      const r = (key >> 16) & 0xff;
      const g = (key >> 8) & 0xff;
      const b = key & 0xff;
      clusters.push({
        id: nextId,
        r,
        g,
        b,
        count,
        active: true,
      });
      nextId += 1;
    });
    return clusters;
  }

  function compressHistogram(histogram, maxEntries) {
    if (histogram.size <= maxEntries) {
      return histogram;
    }

    let shift = 1;
    while (shift <= 7) {
      const buckets = new Map();

      histogram.forEach((count, key) => {
        const r = (key >> 16) & 0xff;
        const g = (key >> 8) & 0xff;
        const b = key & 0xff;
        const br = r >> shift;
        const bg = g >> shift;
        const bb = b >> shift;
        const bucketKey = (br << 16) | (bg << 8) | bb;

        const current = buckets.get(bucketKey) || { sumR: 0, sumG: 0, sumB: 0, count: 0 };
        current.sumR += r * count;
        current.sumG += g * count;
        current.sumB += b * count;
        current.count += count;
        buckets.set(bucketKey, current);
      });

      if (buckets.size <= maxEntries || shift === 7) {
        const compressed = new Map();
        buckets.forEach((value) => {
          const count = Math.max(1, value.count);
          const r = Math.round(value.sumR / count);
          const g = Math.round(value.sumG / count);
          const b = Math.round(value.sumB / count);
          compressed.set(colorKey(r, g, b), count);
        });
        return compressed;
      }

      shift += 1;
    }

    return histogram;
  }

  function mergeCost(a, b) {
    const dr = a.r - b.r;
    const dg = a.g - b.g;
    const db = a.b - b.b;
    const distSq = (dr * dr) + (dg * dg) + (db * db);
    return ((a.count * b.count) / (a.count + b.count)) * distSq;
  }

  function findBestPair(clusters) {
    let bestA = -1;
    let bestB = -1;
    let bestCost = Number.POSITIVE_INFINITY;

    for (let i = 0; i < clusters.length; i += 1) {
      const a = clusters[i];
      if (!a.active) {
        continue;
      }
      for (let j = i + 1; j < clusters.length; j += 1) {
        const b = clusters[j];
        if (!b.active) {
          continue;
        }
        const cost = mergeCost(a, b);
        if (cost < bestCost) {
          bestCost = cost;
          bestA = i;
          bestB = j;
        }
      }
    }

    return { bestA, bestB, bestCost };
  }

  async function reduceClustersPNNLike(clusters, targetCount, onProgress) {
    const totalActiveStart = clusters.length;
    let activeCount = totalActiveStart;
    let mergeSteps = 0;

    while (activeCount > targetCount) {
      const pair = findBestPair(clusters);
      if (pair.bestA < 0 || pair.bestB < 0) {
        break;
      }

      const a = clusters[pair.bestA];
      const b = clusters[pair.bestB];
      const mergedCount = a.count + b.count;

      a.r = Math.round((a.r * a.count + b.r * b.count) / mergedCount);
      a.g = Math.round((a.g * a.count + b.g * b.count) / mergedCount);
      a.b = Math.round((a.b * a.count + b.b * b.count) / mergedCount);
      a.count = mergedCount;

      b.active = false;
      activeCount -= 1;
      mergeSteps += 1;

      if (onProgress && (mergeSteps % 8 === 0 || activeCount === targetCount)) {
        const completed = totalActiveStart - activeCount;
        const total = totalActiveStart - targetCount;
        const ratio = total > 0 ? completed / total : 1;
        onProgress(clamp(ratio, 0, 1));
        await new Promise((resolve) => setTimeout(resolve, 0));
      }
    }

    const palette = [];
    for (let i = 0; i < clusters.length; i += 1) {
      const c = clusters[i];
      if (!c.active) {
        continue;
      }
      palette.push({ r: c.r, g: c.g, b: c.b, count: c.count });
    }

    palette.sort((left, right) => right.count - left.count);
    return palette;
  }

  async function quantizeImageWithPNNLike(imageData, targetCount, options = {}) {
    if (!imageData) {
      throw new Error('ImageData is required for quantization.');
    }

    const histogram = buildHistogram(imageData);
    const uniqueColorCount = histogram.size;
    const safeTarget = clamp(Number(targetCount) || 16, 1, 65535);

    if (safeTarget >= uniqueColorCount) {
      return {
        palette: histogramToClusters(histogram).map((cluster) => ({
          r: cluster.r,
          g: cluster.g,
          b: cluster.b,
          count: cluster.count,
        })),
        uniqueColorCount,
        compressedSourceCount: uniqueColorCount,
      };
    }

    const compressedHistogram = compressHistogram(histogram, options.maxInputColors || MAX_PNN_INPUT_COLORS);
    const clusters = histogramToClusters(compressedHistogram);

    const adjustedTarget = clamp(safeTarget, 1, clusters.length);
    const palette = await reduceClustersPNNLike(clusters, adjustedTarget, options.onProgress);

    return {
      palette,
      uniqueColorCount,
      compressedSourceCount: clusters.length,
    };
  }

  globalScope.PaletteToolQuantizer = {
    quantizeImageWithPNNLike,
  };
})(window);
