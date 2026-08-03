(function attachPaletteToolImageIndex(globalScope) {
  function renderIndexedImageToImageData(width, height, indexImage, palette) {
    if (!indexImage || !palette) {
      throw new Error('Index image and palette are required for rendering.');
    }

    const imageData = new ImageData(width, height);
    const out = imageData.data;

    for (let pixel = 0, outOffset = 0; pixel < indexImage.length; pixel += 1, outOffset += 4) {
      const colorIndex = indexImage[pixel];
      const color = palette[colorIndex] || { r: 0, g: 0, b: 0 };
      out[outOffset] = color.r;
      out[outOffset + 1] = color.g;
      out[outOffset + 2] = color.b;
      out[outOffset + 3] = 255;
    }

    return imageData;
  }

  globalScope.PaletteToolImageIndex = {
    renderIndexedImageToImageData,
  };
})(window);
