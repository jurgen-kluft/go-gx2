(function attachPaletteToolImageIndex(globalScope) {
  function validateSourceImageData(width, height, sourceImageData) {
    if (!sourceImageData || !sourceImageData.data) {
      throw new Error('Source image data is required for rendering.');
    }
    if (sourceImageData.width !== width || sourceImageData.height !== height) {
      throw new Error('Source image dimensions must match the rendered image.');
    }
    if (sourceImageData.data.length !== width * height * 4) {
      throw new Error('Source image data length does not match its dimensions.');
    }
  }

  function renderIndexedImageToImageData(width, height, indexImage, palette, sourceImageData) {
    if (!indexImage || !palette) {
      throw new Error('Index image and palette are required for rendering.');
    }
    if (indexImage.length !== width * height) {
      throw new Error('Index image length does not match the rendered dimensions.');
    }
    validateSourceImageData(width, height, sourceImageData);

    const imageData = new ImageData(width, height);
    const out = imageData.data;
    const source = sourceImageData.data;

    for (let pixel = 0, outOffset = 0; pixel < indexImage.length; pixel += 1, outOffset += 4) {
      const colorIndex = indexImage[pixel];
      const color = palette[colorIndex] || { r: 0, g: 0, b: 0 };
      out[outOffset] = color.r;
      out[outOffset + 1] = color.g;
      out[outOffset + 2] = color.b;
      out[outOffset + 3] = source[outOffset + 3];
    }

    return imageData;
  }

  function renderAlphaToImageData(sourceImageData) {
    if (!sourceImageData) {
      throw new Error('Source image data is required for alpha rendering.');
    }
    validateSourceImageData(sourceImageData.width, sourceImageData.height, sourceImageData);

    const imageData = new ImageData(sourceImageData.width, sourceImageData.height);
    const source = sourceImageData.data;
    const out = imageData.data;

    for (let offset = 0; offset < source.length; offset += 4) {
      const alpha = source[offset + 3];
      out[offset] = alpha;
      out[offset + 1] = alpha;
      out[offset + 2] = alpha;
      out[offset + 3] = 255;
    }

    return imageData;
  }

  globalScope.PaletteToolImageIndex = {
    renderAlphaToImageData,
    renderIndexedImageToImageData,
  };
})(window);
