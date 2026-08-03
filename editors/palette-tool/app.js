const { renderIndexedImageToImageData } = window.PaletteToolImageIndex;
const {
  clamp,
  extractPaletteAndIndexFromImageData,
  paletteColorToCss,
  reindexToPalette,
} = window.PaletteToolUtils;
const { createInitialState, ZOOM_LEVELS } = window.PaletteToolState;
const { quantizeImageWithPNNLike } = window.PaletteToolQuantizer;

const state = createInitialState();

const elements = {
  imageInput: document.getElementById('imageInput'),
  paletteFileInput: document.getElementById('paletteFileInput'),
  currentButton: document.getElementById('currentButton'),
  quantizePanelButton: document.getElementById('quantizePanelButton'),
  currentPanel: document.getElementById('currentPanel'),
  quantizePanel: document.getElementById('quantizePanel'),
  closeCurrentPanelButton: document.getElementById('closeCurrentPanelButton'),
  closeQuantizePanelButton: document.getElementById('closeQuantizePanelButton'),
  chooseImageFileButton: document.getElementById('chooseImageFileButton'),
  choosePaletteFileButton: document.getElementById('choosePaletteFileButton'),
  savePaletteButton: document.getElementById('savePaletteButton'),
  saveIndexedPngButton: document.getElementById('saveIndexedPngButton'),
  extractButton: document.getElementById('extractButton'),
  reindexButton: document.getElementById('reindexButton'),
  quantizeButton: document.getElementById('quantizeButton'),
  status: document.getElementById('status'),
  imageSummary: document.getElementById('imageSummary'),
  actionSummary: document.getElementById('actionSummary'),
  viewModeButton: document.getElementById('viewModeButton'),
  workspace: document.getElementById('workspace'),
  editorWorkspace: document.getElementById('editorWorkspace'),
  previewWorkspace: document.getElementById('previewWorkspace'),
  editorWorkspaceSummary: document.getElementById('editorWorkspaceSummary'),
  canvas: document.getElementById('canvas'),
  zoomSelect: document.getElementById('zoomSelect'),
  zoomOutButton: document.getElementById('zoomOutButton'),
  zoomInButton: document.getElementById('zoomInButton'),
  zoomResetButton: document.getElementById('zoomResetButton'),
  paletteMeta: document.getElementById('paletteMeta'),
  paletteGridMain: document.getElementById('paletteGridMain'),
  colorLimitInput: document.getElementById('colorLimitInput'),
  quantizeCountInput: document.getElementById('quantizeCountInput'),
  quantizeCountRange: document.getElementById('quantizeCountRange'),
  selectionSummary: document.getElementById('selectionSummary'),
  editR: document.getElementById('editR'),
  editG: document.getElementById('editG'),
  editB: document.getElementById('editB'),
  applyColorButton: document.getElementById('applyColorButton'),
  mergeButton: document.getElementById('mergeButton'),
  sortModeSelect: document.getElementById('sortModeSelect'),
  sortPaletteButton: document.getElementById('sortPaletteButton'),
};

const canvasCtx = elements.canvas.getContext('2d');
const sourceCanvas = document.createElement('canvas');
const sourceCtx = sourceCanvas.getContext('2d');

function isPreviewMode() {
  return state.viewMode === 'preview';
}

function closePanels() {
  elements.currentPanel.classList.add('hidden');
  elements.quantizePanel.classList.add('hidden');
}

function togglePanel(panelName) {
  if (panelName === 'current') {
    const open = elements.currentPanel.classList.contains('hidden');
    closePanels();
    if (open) {
      elements.currentPanel.classList.remove('hidden');
    }
    return;
  }

  if (panelName === 'quantize') {
    const open = elements.quantizePanel.classList.contains('hidden');
    closePanels();
    if (open) {
      elements.quantizePanel.classList.remove('hidden');
    }
  }
}

function setViewMode(mode) {
  state.viewMode = mode === 'preview' ? 'preview' : 'editor';
  const preview = isPreviewMode();

  elements.editorWorkspace.classList.toggle('hidden', preview);
  elements.previewWorkspace.classList.toggle('hidden', !preview);
  elements.viewModeButton.textContent = 'View Mode';
  elements.viewModeButton.title = preview ? 'Currently Image View' : 'Currently Palette View';
  elements.viewModeButton.classList.toggle('active', preview);
  elements.zoomSelect.disabled = !hasImage() || !preview;
  elements.zoomOutButton.disabled = !hasImage() || !preview;
  elements.zoomInButton.disabled = !hasImage() || !preview;
  elements.zoomResetButton.disabled = !hasImage() || !preview;

  if (preview) {
    render();
  }
}

function hasImage() {
  return Boolean(state.sourceImageData);
}

function hasIndexedImage() {
  return Boolean(state.hasIndexedImage && state.indexImage && state.palette.length);
}

function setStatus(message, tone = 'neutral') {
  elements.status.textContent = message;
  elements.status.className = `status${tone === 'neutral' ? '' : ` ${tone}`}`;
}

function updateControls() {
  const active = hasImage();
  const hasPalette = hasIndexedImage();
  const preview = isPreviewMode();
  const selectedCount = state.selectedPaletteIndices.length;
  const hasSingleSelection = selectedCount === 1;
  const hasMultiSelection = selectedCount >= 2;

  elements.quantizePanelButton.disabled = !active;
  elements.extractButton.disabled = !active;
  elements.reindexButton.disabled = !hasPalette;
  elements.quantizeButton.disabled = !active;
  elements.choosePaletteFileButton.disabled = false;
  elements.savePaletteButton.disabled = !hasPalette;
  elements.saveIndexedPngButton.disabled = !hasPalette;
  elements.zoomSelect.disabled = !active || !preview;
  elements.zoomOutButton.disabled = !active || !preview;
  elements.zoomInButton.disabled = !active || !preview;
  elements.zoomResetButton.disabled = !active || !preview;
  elements.quantizeCountInput.disabled = !active;
  elements.quantizeCountRange.disabled = !active;
  elements.applyColorButton.disabled = !hasPalette || !hasSingleSelection;
  elements.mergeButton.disabled = !hasPalette || !hasMultiSelection;
  elements.sortPaletteButton.disabled = !hasPalette || state.palette.length < 2;
  elements.sortModeSelect.disabled = !hasPalette;
  elements.editR.disabled = !hasPalette || !hasSingleSelection;
  elements.editG.disabled = !hasPalette || !hasSingleSelection;
  elements.editB.disabled = !hasPalette || !hasSingleSelection;
}

function updateSummary() {
  if (!hasImage()) {
    elements.imageSummary.textContent = 'No image loaded';
    elements.actionSummary.textContent = 'Load a PNG image to extract a working palette.';
    return;
  }

  const indexedLabel = state.hasIndexedImage ? 'Indexed image: ready' : 'Indexed image: pending';
  elements.imageSummary.textContent = `${state.imageName} • ${state.width}x${state.height}`;
  elements.actionSummary.textContent = `${indexedLabel} • Palette colors ${state.palette.length}`;
}

function updateCanvasSize() {
  if (!hasImage()) {
    elements.canvas.width = 960;
    elements.canvas.height = 640;
    elements.canvas.style.width = '960px';
    elements.canvas.style.height = '640px';
    return;
  }

  const width = Math.max(1, Math.round(state.width * state.zoom));
  const height = Math.max(1, Math.round(state.height * state.zoom));
  elements.canvas.width = width;
  elements.canvas.height = height;
  elements.canvas.style.width = `${width}px`;
  elements.canvas.style.height = `${height}px`;
}

function renderPlaceholder() {
  canvasCtx.clearRect(0, 0, elements.canvas.width, elements.canvas.height);
  canvasCtx.fillStyle = '#b5c4d3';
  canvasCtx.font = '600 24px "Avenir Next", sans-serif';
  canvasCtx.textAlign = 'center';
  canvasCtx.fillText('Load a PNG to begin palette extraction', elements.canvas.width / 2, elements.canvas.height / 2 - 10);
  canvasCtx.fillStyle = '#8da2b8';
  canvasCtx.font = '16px "Avenir Next", sans-serif';
  canvasCtx.fillText('This preview is rendered from uint16 index image + active palette.', elements.canvas.width / 2, elements.canvas.height / 2 + 20);
}

function renderIndexedPreview() {
  if (!hasIndexedImage()) {
    return;
  }

  state.previewImageData = renderIndexedImageToImageData(
    state.width,
    state.height,
    state.indexImage,
    state.palette,
  );

  sourceCtx.putImageData(state.previewImageData, 0, 0);
  canvasCtx.setTransform(1, 0, 0, 1, 0, 0);
  canvasCtx.clearRect(0, 0, elements.canvas.width, elements.canvas.height);
  canvasCtx.imageSmoothingEnabled = false;
  canvasCtx.drawImage(sourceCanvas, 0, 0, state.width, state.height, 0, 0, elements.canvas.width, elements.canvas.height);
}

function render() {
  if (!isPreviewMode()) {
    return;
  }

  updateCanvasSize();
  if (!hasImage()) {
    renderPlaceholder();
    return;
  }

  if (hasIndexedImage()) {
    renderIndexedPreview();
    return;
  }

  canvasCtx.clearRect(0, 0, elements.canvas.width, elements.canvas.height);
  sourceCtx.putImageData(state.sourceImageData, 0, 0);
  canvasCtx.imageSmoothingEnabled = false;
  canvasCtx.drawImage(sourceCanvas, 0, 0, state.width, state.height, 0, 0, elements.canvas.width, elements.canvas.height);
}

function createSwatchElement(index, color, selectedSet) {
  const swatch = document.createElement('div');
  swatch.className = 'swatch';
  if (selectedSet.has(index)) {
    swatch.classList.add('selected');
  }
  swatch.style.background = paletteColorToCss(color);
  swatch.title = `#${index} rgb(${color.r}, ${color.g}, ${color.b}) • ${color.count} px`;
  swatch.tabIndex = 0;
  swatch.setAttribute('role', 'button');
  swatch.setAttribute('aria-label', `Palette color ${index}`);
  swatch.addEventListener('click', (event) => {
    onPaletteSwatchSelect(index, event.metaKey || event.ctrlKey);
  });

  const label = document.createElement('span');
  label.textContent = `${index} · ${color.count}`;
  swatch.append(label);
  return swatch;
}

function syncZoomSelect(zoom, customLabel = '') {
  const select = elements.zoomSelect;
  const normalized = Number(zoom);
  const options = Array.from(select.options);
  const exactOption = options.find((option) => Math.abs(Number(option.value) - normalized) < 1e-6 && option.dataset.custom !== 'fit');
  const customOption = select.querySelector('option[data-custom="fit"]');

  if (exactOption) {
    if (customOption) {
      customOption.remove();
    }
    select.value = exactOption.value;
    return;
  }

  const percent = Math.round(normalized * 1000) / 10;
  const optionLabel = customLabel || `${percent}%`;
  const customValue = normalized.toFixed(6);

  if (customOption) {
    customOption.value = customValue;
    customOption.textContent = optionLabel;
  } else {
    const option = document.createElement('option');
    option.dataset.custom = 'fit';
    option.value = customValue;
    option.textContent = optionLabel;
    select.append(option);
  }

  select.value = customValue;
}

function setZoom(zoom, options = {}) {
  const safeZoom = clamp(Number(zoom) || 1, 0.05, 32);
  state.zoom = safeZoom;
  syncZoomSelect(safeZoom, options.customLabel || '');
  render();
}

function fitZoomToWorkspace() {
  if (!hasImage()) {
    return;
  }

  const stagePadding = 32;
  const availableWidth = Math.max(1, elements.workspace.clientWidth - stagePadding);
  const availableHeight = Math.max(1, elements.workspace.clientHeight - stagePadding);
  const fitScale = Math.min(availableWidth / state.width, availableHeight / state.height);
  setZoom(fitScale, { customLabel: `Fit ${Math.round(fitScale * 1000) / 10}%` });
}

function zoomByStep(direction) {
  if (direction > 0) {
    for (const level of ZOOM_LEVELS) {
      if (level > state.zoom + 1e-6) {
        setZoom(level);
        return;
      }
    }
    setZoom(ZOOM_LEVELS[ZOOM_LEVELS.length - 1]);
    return;
  }

  if (direction < 0) {
    for (let i = ZOOM_LEVELS.length - 1; i >= 0; i -= 1) {
      const level = ZOOM_LEVELS[i];
      if (level < state.zoom - 1e-6) {
        setZoom(level);
        return;
      }
    }
    setZoom(ZOOM_LEVELS[0]);
  }
}

function updatePaletteUI() {
  if (!state.palette.length) {
    elements.paletteMeta.textContent = 'Switch between palette editing and image preview.';
    elements.paletteGridMain.replaceChildren();
    elements.selectionSummary.textContent = 'No color selected.';
    elements.editorWorkspaceSummary.textContent = 'Palette editing is prioritized here. Switch to Preview to see the image.';
    return;
  }

  const selectedSet = new Set(state.selectedPaletteIndices);
  const shownCount = Math.min(state.palette.length, state.maxDisplaySwatches);
  const hiddenCount = Math.max(0, state.palette.length - shownCount);
  const extraText = hiddenCount ? ` • Showing ${shownCount}/${state.palette.length}` : '';
  elements.paletteMeta.textContent = `Palette ${state.palette.length} colors${extraText}.`; 
  elements.editorWorkspaceSummary.textContent = `Palette colors ${state.palette.length}. Click swatches to select and edit.`;

  if (!state.selectedPaletteIndices.length) {
    elements.selectionSummary.textContent = 'No color selected.';
  } else if (state.selectedPaletteIndices.length === 1) {
    const selectedIndex = state.selectedPaletteIndices[0];
    const color = state.palette[selectedIndex];
    if (color) {
      elements.selectionSummary.textContent = `Selected #${selectedIndex} rgb(${color.r}, ${color.g}, ${color.b}) • ${color.count} px`;
      elements.editR.value = String(color.r);
      elements.editG.value = String(color.g);
      elements.editB.value = String(color.b);
    }
  } else {
    elements.selectionSummary.textContent = `Selected ${state.selectedPaletteIndices.length} colors.`;
  }

  const mainFragment = document.createDocumentFragment();
  for (let i = 0; i < shownCount; i += 1) {
    const color = state.palette[i];
    mainFragment.append(createSwatchElement(i, color, selectedSet));
  }

  elements.paletteGridMain.replaceChildren(mainFragment);
}

function resetIndexedState() {
  state.palette = [];
  state.indexImage = null;
  state.selectedPaletteIndices = [];
  state.colorCounts = null;
  state.uniqueColorCount = 0;
  state.overflowMappedCount = 0;
  state.hasIndexedImage = false;
}

function clampRgb(value) {
  const parsed = Number.parseInt(value, 10);
  return clamp(Number.isNaN(parsed) ? 0 : parsed, 0, 255);
}

function normalizeSelection() {
  const filtered = state.selectedPaletteIndices
    .filter((index) => index >= 0 && index < state.palette.length)
    .sort((left, right) => left - right);

  const deduped = [];
  let previous = -1;
  for (let i = 0; i < filtered.length; i += 1) {
    const current = filtered[i];
    if (current !== previous) {
      deduped.push(current);
      previous = current;
    }
  }
  state.selectedPaletteIndices = deduped;
}

function onPaletteSwatchSelect(index, append) {
  if (!append) {
    state.selectedPaletteIndices = [index];
  } else {
    const existing = state.selectedPaletteIndices.indexOf(index);
    if (existing >= 0) {
      state.selectedPaletteIndices.splice(existing, 1);
    } else {
      state.selectedPaletteIndices.push(index);
    }
  }

  normalizeSelection();
  updateControls();
  updatePaletteUI();
}

function reindexAfterPaletteMutation(statusMessage, tone = 'ok') {
  state.indexImage = reindexToPalette(state.sourceImageData, state.palette);
  state.hasIndexedImage = true;
  state.overflowMappedCount = 0;
  state.uniqueColorCount = state.palette.length;
  normalizeSelection();
  updateControls();
  updateSummary();
  updatePaletteUI();
  render();
  setStatus(statusMessage, tone);
}

function refreshAfterPaletteColorChange(statusMessage, tone = 'ok') {
  normalizeSelection();
  updateControls();
  updateSummary();
  updatePaletteUI();
  render();
  setStatus(statusMessage, tone);
}

function applySelectedColorEdit() {
  if (!hasIndexedImage() || state.selectedPaletteIndices.length !== 1) {
    return;
  }

  const index = state.selectedPaletteIndices[0];
  const color = state.palette[index];
  if (!color) {
    return;
  }

  color.r = clampRgb(elements.editR.value);
  color.g = clampRgb(elements.editG.value);
  color.b = clampRgb(elements.editB.value);

  refreshAfterPaletteColorChange(`Updated color #${index}. Index assignments preserved.`, 'ok');
}

function mergeSelectedColors() {
  if (!hasIndexedImage() || state.selectedPaletteIndices.length < 2) {
    return;
  }

  const selected = state.selectedPaletteIndices.slice().sort((left, right) => left - right);
  let sumCount = 0;
  let weightedR = 0;
  let weightedG = 0;
  let weightedB = 0;

  for (let i = 0; i < selected.length; i += 1) {
    const color = state.palette[selected[i]];
    const weight = Math.max(1, color.count || 1);
    sumCount += weight;
    weightedR += color.r * weight;
    weightedG += color.g * weight;
    weightedB += color.b * weight;
  }

  const mergedColor = {
    r: Math.round(weightedR / sumCount),
    g: Math.round(weightedG / sumCount),
    b: Math.round(weightedB / sumCount),
    count: sumCount,
  };

  const keepIndex = selected[0];
  state.palette[keepIndex] = mergedColor;

  for (let i = selected.length - 1; i >= 1; i -= 1) {
    state.palette.splice(selected[i], 1);
  }

  state.selectedPaletteIndices = [keepIndex];
  reindexAfterPaletteMutation(`Merged ${selected.length} colors into #${keepIndex} and reindexed.`, 'ok');
}

function colorHue(color) {
  const r = color.r / 255;
  const g = color.g / 255;
  const b = color.b / 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const delta = max - min;

  if (delta === 0) {
    return 0;
  }
  if (max === r) {
    return ((g - b) / delta + (g < b ? 6 : 0)) / 6;
  }
  if (max === g) {
    return ((b - r) / delta + 2) / 6;
  }
  return ((r - g) / delta + 4) / 6;
}

function colorLuminance(color) {
  return (0.2126 * color.r) + (0.7152 * color.g) + (0.0722 * color.b);
}

function sortPaletteWithMode(mode) {
  if (!hasIndexedImage() || state.palette.length < 2) {
    return;
  }

  const sorted = state.palette.slice();
  if (mode === 'usage') {
    sorted.sort((left, right) => (right.count || 0) - (left.count || 0));
  } else if (mode === 'luminance') {
    sorted.sort((left, right) => colorLuminance(left) - colorLuminance(right));
  } else if (mode === 'rgb') {
    sorted.sort((left, right) => {
      if (left.r !== right.r) {
        return left.r - right.r;
      }
      if (left.g !== right.g) {
        return left.g - right.g;
      }
      return left.b - right.b;
    });
  } else {
    sorted.sort((left, right) => {
      const hueDiff = colorHue(left) - colorHue(right);
      if (Math.abs(hueDiff) > 1e-8) {
        return hueDiff;
      }
      return colorLuminance(left) - colorLuminance(right);
    });
  }

  state.palette = sorted;
  state.selectedPaletteIndices = [];
  reindexAfterPaletteMutation(`Sorted palette by ${mode} and reindexed.`, 'ok');
}

function loadImageFromDataUrl(dataUrl, name) {
  const image = new Image();
  image.onload = () => {
    state.imageName = name;
    state.width = image.naturalWidth || image.width;
    state.height = image.naturalHeight || image.height;

    sourceCanvas.width = state.width;
    sourceCanvas.height = state.height;
    sourceCtx.clearRect(0, 0, state.width, state.height);
    sourceCtx.drawImage(image, 0, 0);

    state.sourceImageData = sourceCtx.getImageData(0, 0, state.width, state.height);
    resetIndexedState();

    fitZoomToWorkspace();
    elements.workspace.scrollTo({ top: 0, left: 0, behavior: 'auto' });

    updateControls();
    updateSummary();
    updatePaletteUI();
    render();
    setStatus(`Loaded ${name}. Click "Extract Palette" to build index image.`, 'ok');
  };
  image.onerror = () => {
    setStatus('Image could not be loaded.', 'bad');
  };
  image.src = dataUrl;
}

function loadImageFile(file) {
  if (!file) {
    return;
  }
  const reader = new FileReader();
  reader.onload = () => {
    loadImageFromDataUrl(String(reader.result || ''), file.name);
  };
  reader.onerror = () => {
    setStatus('File could not be read.', 'bad');
  };
  reader.readAsDataURL(file);
}

function channelToHex(value) {
  return clamp(value, 0, 255).toString(16).toUpperCase().padStart(2, '0');
}

function rgbToHex(color) {
  return `#${channelToHex(color.r)}${channelToHex(color.g)}${channelToHex(color.b)}`;
}

function parseHexColor(hex, index) {
  if (typeof hex !== 'string') {
    throw new Error(`Palette color at index ${index} must be a string like #0048BA.`);
  }

  const trimmed = hex.trim();
  const match = /^#([0-9a-fA-F]{6})$/.exec(trimmed);
  if (!match) {
    throw new Error(`Palette color at index ${index} is invalid. Expected #RRGGBB, got "${hex}".`);
  }

  const raw = match[1];
  return {
    r: Number.parseInt(raw.slice(0, 2), 16),
    g: Number.parseInt(raw.slice(2, 4), 16),
    b: Number.parseInt(raw.slice(4, 6), 16),
    count: 0,
  };
}

function savePaletteToFile() {
  if (!state.palette.length) {
    return;
  }

  const payload = state.palette.map((color) => rgbToHex(color));

  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  const baseName = (state.imageName || 'palette').replace(/\.[^.]+$/, '');
  anchor.href = url;
  anchor.download = `${baseName}.palette.json`;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
  setStatus(`Saved ${anchor.download}`, 'ok');
}

function saveIndexedPng() {
  if (!hasIndexedImage()) {
    return;
  }

  const imageData = renderIndexedImageToImageData(
    state.width,
    state.height,
    state.indexImage,
    state.palette,
  );

  sourceCanvas.width = state.width;
  sourceCanvas.height = state.height;
  sourceCtx.putImageData(imageData, 0, 0);

  sourceCanvas.toBlob((blob) => {
    if (!blob) {
      setStatus('Indexed PNG export failed.', 'bad');
      return;
    }

    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    const baseName = (state.imageName || 'indexed').replace(/\.[^.]+$/, '');
    anchor.href = url;
    anchor.download = `${baseName}.indexed.png`;
    document.body.append(anchor);
    anchor.click();
    anchor.remove();
    URL.revokeObjectURL(url);
    setStatus(`Saved ${anchor.download}`, 'ok');
  }, 'image/png');
}

function applyLoadedPaletteData(paletteData, sourceName) {
  if (!Array.isArray(paletteData)) {
    throw new Error('Strict mode: palette JSON must be an array of hex strings like ["#0048BA"].');
  }
  if (!paletteData.length) {
    throw new Error('Palette file contains no colors.');
  }

  const normalized = new Array(paletteData.length);
  for (let i = 0; i < paletteData.length; i += 1) {
    normalized[i] = parseHexColor(paletteData[i], i);
  }

  state.palette = normalized;
  state.selectedPaletteIndices = [];
  if (hasImage()) {
    reindexAfterPaletteMutation(`Loaded palette from ${sourceName} and reindexed.`, 'ok');
    return;
  }

  state.hasIndexedImage = false;
  updateControls();
  updateSummary();
  updatePaletteUI();
  setStatus(`Loaded palette from ${sourceName}.`, 'ok');
}

function loadPaletteFile(file) {
  if (!file) {
    return;
  }

  const reader = new FileReader();
  reader.onload = () => {
    try {
      const raw = JSON.parse(String(reader.result || '{}'));
      applyLoadedPaletteData(raw, file.name);
    } catch (error) {
      setStatus(`Palette file could not be loaded: ${error.message}`, 'bad');
    }
  };
  reader.onerror = () => {
    setStatus('Palette file could not be read.', 'bad');
  };
  reader.readAsText(file);
}

function getColorLimit() {
  const parsed = Number.parseInt(elements.colorLimitInput.value, 10);
  return clamp(Number.isNaN(parsed) ? 65535 : parsed, 1, 65535);
}

function getQuantizeTarget() {
  const parsed = Number.parseInt(elements.quantizeCountInput.value, 10);
  return clamp(Number.isNaN(parsed) ? 256 : parsed, 2, 512);
}

function syncQuantizeInputs(source) {
  const value = source === 'range'
    ? clamp(Number.parseInt(elements.quantizeCountRange.value, 10) || 256, 2, 512)
    : clamp(Number.parseInt(elements.quantizeCountInput.value, 10) || 256, 2, 512);

  elements.quantizeCountInput.value = String(value);
  elements.quantizeCountRange.value = String(value);
}

function extractPaletteFromCurrentImage() {
  if (!hasImage()) {
    return;
  }

  try {
    const colorLimit = getColorLimit();
    const result = extractPaletteAndIndexFromImageData(state.sourceImageData, {
      maxPaletteSize: colorLimit,
    });

    state.palette = result.palette;
    state.indexImage = result.indexImage;
    state.uniqueColorCount = result.uniqueColorCount;
    state.overflowMappedCount = result.overflowMappedCount;
    state.hasIndexedImage = true;

    updateControls();
    updateSummary();
    updatePaletteUI();
    render();

    const overflowNote = result.overflowMappedCount
      ? ` ${result.overflowMappedCount} pixels mapped to nearest colors after limit ${colorLimit}.`
      : '';

    setStatus(
      `Palette extracted: ${state.palette.length} colors, ${result.uniqueColorCount} unique source colors.${overflowNote}`,
      result.overflowMappedCount ? 'warn' : 'ok',
    );
  } catch (error) {
    setStatus(`Palette extraction failed: ${error.message}`, 'bad');
  }
}

function reindexWithCurrentPalette() {
  if (!hasImage() || !state.palette.length) {
    return;
  }

  try {
    reindexAfterPaletteMutation('Re-indexed from source to current palette.', 'ok');
  } catch (error) {
    setStatus(`Reindex failed: ${error.message}`, 'bad');
  }
}

async function quantizeCurrentImage() {
  if (!hasImage()) {
    return;
  }

  const targetCount = getQuantizeTarget();
  setStatus(`Quantizing to ${targetCount} colors...`, 'warn');
  elements.quantizeButton.disabled = true;

  try {
    const result = await quantizeImageWithPNNLike(state.sourceImageData, targetCount, {
      onProgress(progress) {
        const percent = Math.round(progress * 100);
        setStatus(`Quantizing to ${targetCount} colors... ${percent}%`, 'warn');
      },
      maxInputColors: 2048,
    });

    state.palette = result.palette;
    state.indexImage = reindexToPalette(state.sourceImageData, state.palette);
    state.uniqueColorCount = result.uniqueColorCount;
    state.overflowMappedCount = 0;
    state.hasIndexedImage = true;
    state.selectedPaletteIndices = [];

    updateControls();
    updateSummary();
    updatePaletteUI();
    render();

    const compressedNote = result.compressedSourceCount < result.uniqueColorCount
      ? ` Source colors were compressed from ${result.uniqueColorCount} to ${result.compressedSourceCount} before merge.`
      : '';

    setStatus(
      `Quantization complete: ${state.palette.length} colors.${compressedNote}`,
      'ok',
    );
  } catch (error) {
    setStatus(`Quantization failed: ${error.message}`, 'bad');
  } finally {
    updateControls();
  }
}

function onDragOver(event) {
  event.preventDefault();
  elements.workspace.classList.add('drop');
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'copy';
  }
}

function onDragLeave(event) {
  if (event.currentTarget === event.target || !elements.workspace.contains(event.relatedTarget)) {
    elements.workspace.classList.remove('drop');
  }
}

function onDrop(event) {
  event.preventDefault();
  elements.workspace.classList.remove('drop');
  const files = event.dataTransfer && event.dataTransfer.files;
  if (!files || !files.length) {
    return;
  }
  loadImageFile(files[0]);
}

function registerEvents() {
  elements.imageInput.addEventListener('change', (event) => {
    const file = event.target.files && event.target.files[0];
    loadImageFile(file);
    event.target.value = '';
  });
  elements.paletteFileInput.addEventListener('change', (event) => {
    const file = event.target.files && event.target.files[0];
    loadPaletteFile(file);
    event.target.value = '';
  });

  elements.currentButton.addEventListener('click', () => {
    togglePanel('current');
  });
  elements.quantizePanelButton.addEventListener('click', () => {
    togglePanel('quantize');
  });
  elements.closeCurrentPanelButton.addEventListener('click', closePanels);
  elements.closeQuantizePanelButton.addEventListener('click', closePanels);
  elements.chooseImageFileButton.addEventListener('click', () => {
    elements.imageInput.click();
  });
  elements.choosePaletteFileButton.addEventListener('click', () => {
    elements.paletteFileInput.click();
  });
  elements.savePaletteButton.addEventListener('click', savePaletteToFile);
  elements.saveIndexedPngButton.addEventListener('click', saveIndexedPng);

  elements.extractButton.addEventListener('click', extractPaletteFromCurrentImage);
  elements.reindexButton.addEventListener('click', reindexWithCurrentPalette);
  elements.quantizeButton.addEventListener('click', () => {
    quantizeCurrentImage();
  });
  elements.quantizeCountInput.addEventListener('input', () => {
    syncQuantizeInputs('input');
  });
  elements.quantizeCountRange.addEventListener('input', () => {
    syncQuantizeInputs('range');
  });
  elements.applyColorButton.addEventListener('click', applySelectedColorEdit);
  elements.mergeButton.addEventListener('click', mergeSelectedColors);
  elements.sortPaletteButton.addEventListener('click', () => {
    sortPaletteWithMode(elements.sortModeSelect.value);
  });

  elements.zoomSelect.addEventListener('change', () => {
    setZoom(Number(elements.zoomSelect.value));
  });
  elements.zoomInButton.addEventListener('click', () => zoomByStep(1));
  elements.zoomOutButton.addEventListener('click', () => zoomByStep(-1));
  elements.zoomResetButton.addEventListener('click', () => {
    if (!hasImage()) {
      setZoom(1);
      return;
    }
    fitZoomToWorkspace();
    elements.workspace.scrollTo({ top: 0, left: 0, behavior: 'auto' });
  });

  elements.workspace.addEventListener('dragover', onDragOver);
  elements.workspace.addEventListener('dragleave', onDragLeave);
  elements.workspace.addEventListener('drop', onDrop);
  window.addEventListener('dragover', onDragOver);
  window.addEventListener('drop', onDrop);

  elements.viewModeButton.addEventListener('click', () => {
    setViewMode(isPreviewMode() ? 'editor' : 'preview');
  });

  window.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      closePanels();
    }
  });
}

function init() {
  registerEvents();
  sourceCanvas.width = 1;
  sourceCanvas.height = 1;
  updateControls();
  syncQuantizeInputs('input');
  updateSummary();
  updatePaletteUI();
  setViewMode('editor');
  setStatus('Load a PNG to begin.', 'neutral');
}

init();
