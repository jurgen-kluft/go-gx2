# go-gx2 Editors

This directory contains three browser-based tools for preparing image assets. They run entirely in the browser, require no build step, and do not upload files anywhere. Open an HTML file directly in a current browser to start using it.

Work is not saved automatically. Download the generated PNG or JSON before closing or reloading a tool.

| Tool | Use it for | Input | Output |
| --- | --- | --- | --- |
| [Palette Tool](PaletteTool.html) | Extracting, editing, reducing, and applying RGB palettes | PNG, palette JSON | `<name>.palette.json`, `<name>.indexed.png` |
| [Sprite Rectangle Editor](SpriteEditor.html) | Defining sprite rectangles on an image | PNG/JPG, sprite JSON | `<name>.sprites.json` |
| [Image Tool](ImageTool.html) | Resizing, cropping, removing backgrounds, and converting anti-aliasing to alpha | PNG/JPG | `<name>.transparent.png` |

## Palette Tool

Open [PaletteTool.html](PaletteTool.html) to extract a palette from a PNG, edit that palette, and re-index the image to the edited colors. The scripts in `palette-tool/` are part of this tool, not a separate editor.

### Basic workflow

1. Select **Current**, then **Choose Image File** and load a PNG.
2. Set **Palette limit** and select **Extract Palette**. The limit may be from 1 to 65,535 colors.
3. Optionally edit, merge, sort, or quantize the palette.
4. Switch between the palette workspace and image preview to inspect the result.
5. Select **Save Palette** or **Save Indexed PNG**.

The palette editor supports the following operations:

- Select a swatch to edit its RGB channels. Use Cmd/Ctrl+click to select multiple swatches for merging.
- Sort by usage, hue, luminance, or RGB. Sorting also updates the image indices.
- Re-index the source image to the nearest colors in the current palette.
- Quantize the source image to between 2 and 512 colors.
- Preview at 50%, 100%, 200%, 400%, or 800% zoom.

### Palette JSON

**Choose Palette File** loads a JSON array containing strict six-digit RGB hex strings. Loading a palette while an image is open immediately re-indexes the image to that palette.

```json
[
	"#000000",
	"#4A90E2",
	"#FFFFFF"
]
```

**Save Palette** writes the same format to `<image-name>.palette.json`. Array position is the color index.

### Alpha behavior

Palette extraction and nearest-color matching use RGB only; source alpha is ignored. **Save Indexed PNG** renders every output pixel as fully opaque. Use the Image Tool when alpha must be cleaned up, and keep the original alpha-bearing image if it is needed later in the asset pipeline.

## Sprite Rectangle Editor

Open [SpriteEditor.html](SpriteEditor.html) to draw and maintain named sprite rectangles over a PNG or JPG image.

### Manual workflow

1. Load an image, or load a previously saved sprite-list JSON file that references an image.
2. Drag over empty image space to create a rectangle.
3. Select a rectangle to edit its name, alpha format, position, and size.
4. Move or resize rectangles on the canvas or with the keyboard.
5. Select **Save JSON** to download `<image-name>.sprites.json`.

The editor provides these controls:

- Drag inside a selected rectangle to move it; drag a highlighted handle to resize it.
- Use arrow keys to move a rectangle by one pixel.
- Use Shift+arrow to extend the corresponding side by one pixel.
- Use Cmd/Ctrl+C and Cmd/Ctrl+V, or **Copy** and **Paste**, to duplicate a rectangle.
- Use Delete or Backspace to remove the selected rectangle.
- Use Alt+mouse wheel or the zoom controls for close editing.
- Hold Space and drag, or drag with the middle mouse button, to pan.

### Grid workflows

There are two ways to create repeated sprite rectangles:

- **Reference expansion:** Cmd/Ctrl+click exactly three sprites in this order: the main sprite, a right-hand reference, and a downward reference. Press E to expand the spacing into a grid.
- **Grid Tool:** Choose row and column counts, optionally provide cell dimensions, then create a grid over the whole image or select a grid area. Enable grid editing to adjust individual lines and select **Extract Sprites** to detect sprite bounds within each cell.

Grid extraction uses image-content heuristics. Review the generated rectangles, especially for sprites with faint, transparent, or complex edges.

### Sprite-list JSON

Saved files use this structure:

```json
{
	"version": 1,
	"image": {
		"name": "buttons.png",
		"path": "",
		"width": 256,
		"height": 128
	},
	"sprites": [
		{
			"id": 1,
			"name": "button_normal",
			"alpha": 0,
			"x": 0,
			"y": 0,
			"width": 32,
			"height": 16
		}
	]
}
```

The loader requires a top-level `sprites` array and numeric `x`, `y`, `width`, and `height` values. Missing names and IDs receive defaults. Alpha accepts `0` (None), `1` (Mask), `2` (A2), `4` (A4), or `8` (A8), and defaults to None. The optional `image.path` or `image.dataUrl` fields can be used when loading a sprite list; otherwise, load the source image separately.

## Image Tool

Open [ImageTool.html](ImageTool.html) to resize or crop an image, remove connected background regions, or convert anti-aliased artwork into an alpha-masked image.

### Resize workflow

1. Enter the target width or height. **Lock aspect ratio** updates the other dimension automatically.
2. Select **Resize** to apply high-quality interpolation.

The target width and height must be positive whole numbers and cannot exceed the current image dimensions. At least one dimension must be smaller, so the tool never enlarges an image. Resizing can be undone.

### Crop workflow

1. Select **Select Crop Area**.
2. Drag a rectangle over the part of the image to keep.
3. Review the selection dimensions and origin, then select **Crop**.

Use **Clear** to draw the selection again, or **Cancel Selection** to return to flood-fill mode. Cropping can be undone and may be combined with resizing in either order.

After creating a crop rectangle, adjust its edges one pixel at a time with the keyboard:

- Left/Right decreases or increases the left edge position.
- Shift+Left/Right decreases or increases the right edge position.
- Up/Down decreases or increases the top edge position.
- Shift+Up/Down decreases or increases the bottom edge position.

Edges remain inside the image and cannot cross, so the crop rectangle stays at least one pixel wide and high.

### Flood-fill workflow

1. Load a PNG or JPG.
2. Set **Tolerance** from 0 (strict) to 150 (loose).
3. Click a pixel to make the connected region of similar color and alpha fully transparent.
4. Keep **Mask** enabled to highlight transparent pixels while checking the result.
5. Select **Save PNG** to download `<image-name>.transparent.png`.

Start with a low tolerance and increase it gradually. JPG compression and uneven backgrounds commonly require a higher value. **Undo** keeps up to 15 image snapshots, including changes to image dimensions. **Reset Image** discards all edits and restores the loaded image at its original dimensions.

### Anti-alias to alpha

**Anti-alias to Alpha** is intended for artwork drawn between known opaque and background colors, such as black artwork anti-aliased against white.

1. Choose the intended opaque color and transparent/background color.
2. Swap them if necessary.
3. Select **Apply to Image**.

The operation changes every nontransparent pixel to the opaque RGB color and derives its alpha from its position between the two selected colors. It is a whole-image conversion, not a local brush operation. Undo or reset if the selected colors do not produce the expected result.

## Suggested Workflow

For an asset that needs all three operations:

1. Resize, crop, or clean the source image with the Image Tool and save a PNG.
2. Use the Palette Tool when an RGB palette or opaque indexed preview is needed.
3. Define named sprite bounds with the Sprite Rectangle Editor.
4. Preserve the downloaded source and JSON files; browser sessions are not project files.

The JSON files produced by these editors describe their own working data. They are not direct `spritepak` command configuration files and may need to be transformed or referenced by a separate asset-building step.
