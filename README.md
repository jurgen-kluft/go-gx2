# gx2 tools

## TGA/PNG to Sprite-Pack

A golang tool to convert TGA/PNG image files to a custom binary file format that can be loaded by the library as an array of images/sprites.

Tool takes a .json file as input that describes the images to be included in the sprite pack and outputs a .bin file that contains the sprite data in the custom binary file format.

```json
{
  "files": [
    {
      "file": "sprite1.png",
      "sprites": [
         {
           "name": "sprite1",
           "format": "RGB565",
           "alpha": "A1"
         }
      ]
    },
    {
      "file": "sprite2.tga",
      "sprites": [
         {
           "name":"sprite2",
           "format": "RGB565",
           "alpha": "A1",
           "rect": {
               "x": 0,
               "y": 0,
               "w": 64,
               "h": 64
            }
         }
      ]
    }
  ]
}
```

- Sprite Pack File Format
  - Header
    - u64 offset to sprite array
    - u32 image count
    - sprite array[]
      - u16 width
      - u16 height
      - u16 format (e.g. RGBA5551, RGBA8888, 8-Bit color palette, 1-Bit, etc..)
      - u16 alpha format (e.g. A8, A4, A1, A0 etc..) (for formats with separate alpha data)
      - u32 pixel data size
      - u64 pixel data offset in file
      - u32 alpha data size
      - u64 alpha data offset in file (for formats with separate alpha data)
      - u64 color palette offset in file (for paletted formats)
    - Data
      - data
    - Color Palette Data
      - raw color palette data for each image (for paletted formats)

## Font Pack

- Font Rendering: https://github.com/mcufont/mcufont
- Font Conversion: https://github.com/erkkah/tigrfont

Tool takes a .json file as input that describes the fonts to be included in the font pack and outputs a .bin file that contains the font and glyph data in the custom binary file format.

```json
{
  "files": [
    {
      "file": "font1.ttf",
      "fonts": [
         {
           "name": "font1x16",
           "size": 16,
           "chars": [
              "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
              "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
              "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
              "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
              "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", " ", 
              "%", "-", "+", ".", "°"
           ]
         },
         {
           "name": "font1x32",
           "size": 32,
           "chars": [
              "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
              "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
              "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
              "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
              "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", " ", 
              "%", "-", "+", ".", "°"
           ]
         }
      ]
    }
  ]
}
```

