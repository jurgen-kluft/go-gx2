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

## Sprite Pack File Format
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

A tool written in Golang that takes a .json file as input (see below) that describes the fonts to be included, together with the ASCII character map, in the font pack and outputs a .bin file that contains the font and glyph data in a custom binary file format.

Each glyph can be of any width and height, with variable spacing and kerning. The font pack file format includes metadata for each glyph, such as its width, height, ascent, descent, and advance, as well as the pixel data for the glyph itself.

## C++ library

The font pack is loaded by the C/C++ library as a `font_context_t` struct that contains an array of `font_t` structs, each of which contains an array of `glyph_t` structs and a character map that maps ASCII characters to glyph indices.
The pointers that are in the structs are written as offsets in the file, and are converted to actual pointers when the font pack is loaded into memory.

```c++
struct glyph_t
{
    i16       advance_x;  // how much to move the pen horizontally to the next character after drawing this one
    i16       bearing_x;  // horizontal distance from the pen position to the left edge of the glyph bitmap
    i16       bearing_y;  // vertical distance from the pen position to the top edge of the glyph bitmap (can be negative)
    u16       width;      // width of the glyph bitmap in pixels
    u16       height;     // height of the glyph bitmap in pixels
};

struct font_t
{
    glyph_t*   glyphs;    // array of glyphs (indexed by glyph index)
    const u8** chars;     // array of bitmaps (indexed by glyph index)
    u8         map[256];  // maps ASCII glyph index (glyph/chars) array (0xFF = char is not supported)
    i16        ascent;    // distance from baseline to top of font
    i16        descent;   // distance from baseline to bottom of font (negative value)
    i16        line_gap;  // distance from bottom of one line to top of next line (can be negative)
    i16        reserved;  // reserved for future use, and makes sure the struct size is aligned to 8 bytes
};

// Maps directly to the font pack file format
struct font_context_t
{
    u32     font_count;
    u32     reserved;
    font_t* fonts;
};
```


```json
{
    "files": [
        {
            "file": "path/to/font1.ttf",
            "fonts": [
                {
                    "name": "font1x16",
                    "size": 16,
                    "chars": {
                        "a": "a",
                        "b": "b",
                        "c": "c",
                        "d": "d",
                        "e": "e",
                        "f": "f",
                        "g": "g",
                        "h": "h",
                        "i": "i",
                        "j": "j",
                        "k": "k",
                        "l": "l",
                        "m": "m",
                        "n": "n",
                        "o": "o",
                        "p": "p",
                        "q": "q",
                        "r": "r",
                        "s": "s",
                        "t": "t",
                        "u": "u",
                        "v": "v",
                        "w": "w",
                        "x": "x",
                        "y": "y",
                        "z": "z",
                        "A": "A",
                        "B": "B",
                        "C": "C",
                        "D": "D",
                        "E": "E",
                        "F": "F",
                        "G": "G",
                        "H": "H",
                        "I": "I",
                        "J": "J",
                        "K": "K",
                        "L": "L",
                        "M": "M",
                        "N": "N",
                        "O": "O",
                        "P": "P",
                        "Q": "Q",
                        "R": "R",
                        "S": "S",
                        "T": "T",
                        "U": "U",
                        "V": "V",
                        "W": "W",
                        "X": "X",
                        "Y": "Y",
                        "Z": "Z",
                        "0": "0",
                        "1": "1",
                        "2": "2",
                        "3": "3",
                        "4": "4",
                        "5": "5",
                        "6": "6",
                        "7": "7",
                        "8": "8",
                        "9": "9",
                        " ": " ",
                        "%": "%",
                        "-": "-",
                        "+": "+",
                        ".": ".",
                        "°": "°"
                    }
                },
                {
                    "name": "path/to/font1x32",
                    "size": 32,
                    "chars": {
                        "a": "a",
                        "b": "b",
                        "c": "c",
                        "d": "d",
                        "e": "e",
                        "f": "f",
                        "g": "g",
                        "h": "h",
                        "i": "i",
                        "j": "j",
                        "k": "k",
                        "l": "l",
                        "m": "m",
                        "n": "n",
                        "o": "o",
                        "p": "p",
                        "q": "q",
                        "r": "r",
                        "s": "s",
                        "t": "t",
                        "u": "u",
                        "v": "v",
                        "w": "w",
                        "x": "x",
                        "y": "y",
                        "z": "z",
                        "A": "A",
                        "B": "B",
                        "C": "C",
                        "D": "D",
                        "E": "E",
                        "F": "F",
                        "G": "G",
                        "H": "H",
                        "I": "I",
                        "J": "J",
                        "K": "K",
                        "L": "L",
                        "M": "M",
                        "N": "N",
                        "O": "O",
                        "P": "P",
                        "Q": "Q",
                        "R": "R",
                        "S": "S",
                        "T": "T",
                        "U": "U",
                        "V": "V",
                        "W": "W",
                        "X": "X",
                        "Y": "Y",
                        "Z": "Z",
                        "0": "0",
                        "1": "1",
                        "2": "2",
                        "3": "3",
                        "4": "4",
                        "5": "5",
                        "6": "6",
                        "7": "7",
                        "8": "8",
                        "9": "9",
                        " ": " ",
                        "%": "%",
                        "-": "-",
                        "+": "+",
                        ".": ".",
                        "°": "°"
                    }
                }
            ]
        }
    ]
}
```

