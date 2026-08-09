(function runImageIndexTests() {
  const { renderAlphaToImageData, renderIndexedImageToImageData } = window.PaletteToolImageIndex;
  const results = document.getElementById('results');
  let failures = 0;

  function appendResult(name, passed, error) {
    const row = document.createElement('p');
    row.className = passed ? 'pass' : 'fail';
    row.textContent = passed ? `PASS ${name}` : `FAIL ${name}: ${error.message}`;
    results.append(row);
  }

  function test(name, callback) {
    try {
      callback();
      appendResult(name, true);
    } catch (error) {
      failures += 1;
      appendResult(name, false, error);
    }
  }

  function assertArrayEqual(actual, expected) {
    const actualValues = Array.from(actual);
    if (actualValues.length !== expected.length) {
      throw new Error(`Expected ${expected.length} values, received ${actualValues.length}`);
    }
    for (let index = 0; index < expected.length; index += 1) {
      if (actualValues[index] !== expected[index]) {
        throw new Error(`Expected ${expected[index]} at ${index}, received ${actualValues[index]}`);
      }
    }
  }

  function assertThrows(callback, messagePart) {
    try {
      callback();
    } catch (error) {
      if (!String(error.message).includes(messagePart)) {
        throw new Error(`Expected error containing "${messagePart}", received "${error.message}"`);
      }
      return;
    }
    throw new Error('Expected function to throw');
  }

  function createMixedAlphaSource() {
    const source = new ImageData(3, 1);
    source.data.set([
      1, 2, 3, 0,
      4, 5, 6, 128,
      7, 8, 9, 255,
    ]);
    return source;
  }

  test('reduced RGB preserves exact source alpha', () => {
    const rendered = renderIndexedImageToImageData(
      3,
      1,
      new Uint16Array([0, 1, 0]),
      [{ r: 10, g: 20, b: 30 }, { r: 40, g: 50, b: 60 }],
      createMixedAlphaSource(),
    );

    assertArrayEqual(rendered.data, [
      10, 20, 30, 0,
      40, 50, 60, 128,
      10, 20, 30, 255,
    ]);
  });

  test('alpha renders as opaque grayscale', () => {
    const rendered = renderAlphaToImageData(createMixedAlphaSource());
    assertArrayEqual(rendered.data, [
      0, 0, 0, 255,
      128, 128, 128, 255,
      255, 255, 255, 255,
    ]);
  });

  test('index dimensions must match', () => {
    assertThrows(
      () => renderIndexedImageToImageData(
        3,
        1,
        new Uint16Array([0, 1]),
        [{ r: 0, g: 0, b: 0 }],
        createMixedAlphaSource(),
      ),
      'Index image length',
    );
  });

  test('source dimensions must match', () => {
    assertThrows(
      () => renderIndexedImageToImageData(
        2,
        1,
        new Uint16Array([0, 0]),
        [{ r: 0, g: 0, b: 0 }],
        createMixedAlphaSource(),
      ),
      'Source image dimensions',
    );
  });

  test('source data length must match', () => {
    assertThrows(
      () => renderAlphaToImageData({ width: 1, height: 1, data: new Uint8ClampedArray(3) }),
      'Source image data length',
    );
  });

  const summary = document.createElement('strong');
  summary.className = failures ? 'fail' : 'pass';
  summary.textContent = failures ? `${failures} test(s) failed` : 'All tests passed';
  results.prepend(summary);
  document.title = failures ? 'FAIL - Palette Tool Image Index Tests' : 'PASS - Palette Tool Image Index Tests';
})();
