(function attachPaletteToolState(globalScope) {
  const ZOOM_LEVELS = [0.5, 1, 2, 4, 8];

  function createInitialState() {
    return {
      imageName: '',
      width: 0,
      height: 0,
      zoom: 1,
      viewMode: 'editor',
      maxDisplaySwatches: 320,
      sourceImageData: null,
      previewImageData: null,
      palette: [],
      indexImage: null,
      selectedPaletteIndices: [],
      colorCounts: null,
      uniqueColorCount: 0,
      overflowMappedCount: 0,
      hasIndexedImage: false,
    };
  }

  globalScope.PaletteToolState = {
    ZOOM_LEVELS,
    createInitialState,
  };
})(window);
