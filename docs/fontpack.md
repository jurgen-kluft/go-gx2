## Font Pack

- Font Rendering: https://github.com/mcufont/mcufont
- Font Conversion: https://github.com/erkkah/tigrfont

A tool written in Golang that takes a .json file as input (see below) that describes the fonts to be included, together with the (extended) ASCII character map, in the font pack and outputs a .bin file that contains the font and glyph data in a custom binary file format.

Each glyph can be of any width and height, with variable spacing and kerning. The font pack file format includes metadata for each glyph, such as its width, height, ascent, descent, and advance, as well as the pixel data for the glyph itself. The pixel data is the minimal width and height bounding box of the glyph, and the metadata includes the bearing (offset) from the pen position to the top-left corner of the bitmap, as well as the advance (how much to move the pen horizontally to the next character after drawing this one).

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
    u8         map[256];  // maps (extended) ASCII to glyph index (index 0xFF -> no glyph)
    i16        ascent;    // distance from baseline to top of font
    i16        descent;   // distance from baseline to bottom of font (negative value)
    i16        line_gap;  // distance from bottom of one line to top of next line (can be negative)
    i16        reserved;  // reserved for future use, and makes sure the struct size is aligned to 8 bytes
};

// Maps directly to the font pack file format
struct font_context_t
{
    font_t* fonts;
    u32     font_count;
    u32     reserved;
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
                    "dpi": 72,
                    "chars": [
                        { "ascii" : "a", "glyph" : "a"} ,
                        { "ascii" : "b", "glyph" : "b"},
                        { "ascii" : "c", "glyph" : "c"},
                        { "ascii" : "d", "glyph" : "d"},
                        { "ascii" : "e", "glyph" : "e"},
                        { "ascii" : "f", "glyph" : "f"},
                        { "ascii" : "g", "glyph" : "g"},
                        { "ascii" : "h", "glyph" : "h"},
                        { "ascii" : "i", "glyph" : "i"},
                        { "ascii" : "j", "glyph" : "j"},
                        { "ascii" : "k", "glyph" : "k"},
                        { "ascii" : "l", "glyph" : "l"},
                        { "ascii" : "m", "glyph" : "m"},
                        { "ascii" : "n", "glyph" : "n"},
                        { "ascii" : "o", "glyph" : "o"},
                        { "ascii" : "p", "glyph" : "p"},
                        { "ascii" : "q", "glyph" : "q"},
                        { "ascii" : "r", "glyph" : "r"},
                        { "ascii" : "s", "glyph" : "s"},
                        { "ascii" : "t", "glyph" : "t"},
                        { "ascii" : "u", "glyph" : "u"},
                        { "ascii" : "v", "glyph" : "v"},
                        { "ascii" : "w", "glyph" : "w"},
                        { "ascii" : "x", "glyph" : "x"},
                        { "ascii" : "y", "glyph" : "y"},
                        { "ascii" : "z", "glyph" : "z"},
                        { "ascii" : "A", "glyph" : "A"},
                        { "ascii" : "B", "glyph" : "B"},
                        { "ascii" : "C", "glyph" : "C"},
                        { "ascii" : "D", "glyph" : "D"},
                        { "ascii" : "E", "glyph" : "E"},
                        { "ascii" : "F", "glyph" : "F"},
                        { "ascii" : "G", "glyph" : "G"},
                        { "ascii" : "H", "glyph" : "H"},
                        { "ascii" : "I", "glyph" : "I"},
                        { "ascii" : "J", "glyph" : "J"},
                        { "ascii" : "K", "glyph" : "K"},
                        { "ascii" : "L", "glyph" : "L"},
                        { "ascii" : "M", "glyph" : "M"},
                        { "ascii" : "N", "glyph" : "N"},
                        { "ascii" : "O", "glyph" : "O"},
                        { "ascii" : "P", "glyph" : "P"},
                        { "ascii" : "Q", "glyph" : "Q"},
                        { "ascii" : "R", "glyph" : "R"},
                        { "ascii" : "S", "glyph" : "S"},
                        { "ascii" : "T", "glyph" : "T"},
                        { "ascii" : "U", "glyph" : "U"},
                        { "ascii" : "V", "glyph" : "V"},
                        { "ascii" : "W", "glyph" : "W"},
                        { "ascii" : "X", "glyph" : "X"},
                        { "ascii" : "Y", "glyph" : "Y"},
                        { "ascii" : "Z", "glyph" : "Z"},
                        { "ascii" : "0", "glyph" : "0"},
                        { "ascii" : "1", "glyph" : "1"},
                        { "ascii" : "2", "glyph" : "2"},
                        { "ascii" : "3", "glyph" : "3"},
                        { "ascii" : "4", "glyph" : "4"},
                        { "ascii" : "5", "glyph" : "5"},
                        { "ascii" : "6", "glyph" : "6"},
                        { "ascii" : "7", "glyph" : "7"},
                        { "ascii" : "8", "glyph" : "8"},
                        { "ascii" : "9", "glyph" : "9"},
                        { "ascii" : " ", "glyph" : " "},
                        { "ascii" : "%", "glyph" : "%"},
                        { "ascii" : "-", "glyph" : "-"},
                        { "ascii" : "+", "glyph" : "+"},
                        { "ascii" : ".", "glyph" : "."},
                        { "ascii" : "°", "glyph" : "°"}
                    ]
                },
                {
                    "name": "path/to/font1x32",
                    "size": 32,
                    "dpi": 72,
                    "chars": [
                        { "ascii" : "a", "glyph" : "a"},
                        { "ascii" : "b", "glyph" : "b"},
                        { "ascii" : "c", "glyph" : "c"},
                        { "ascii" : "d", "glyph" : "d"},
                        { "ascii" : "e", "glyph" : "e"},
                        { "ascii" : "f", "glyph" : "f"},
                        { "ascii" : "g", "glyph" : "g"},
                        { "ascii" : "h", "glyph" : "h"},
                        { "ascii" : "i", "glyph" : "i"},
                        { "ascii" : "j", "glyph" : "j"},
                        { "ascii" : "k", "glyph" : "k"},
                        { "ascii" : "l", "glyph" : "l"},
                        { "ascii" : "m", "glyph" : "m"},
                        { "ascii" : "n", "glyph" : "n"},
                        { "ascii" : "o", "glyph" : "o"},
                        { "ascii" : "p", "glyph" : "p"},
                        { "ascii" : "q", "glyph" : "q"},
                        { "ascii" : "r", "glyph" : "r"},
                        { "ascii" : "s", "glyph" : "s"},
                        { "ascii" : "t", "glyph" : "t"},
                        { "ascii" : "u", "glyph" : "u"},
                        { "ascii" : "v", "glyph" : "v"},
                        { "ascii" : "w", "glyph" : "w"},
                        { "ascii" : "x", "glyph" : "x"},
                        { "ascii" : "y", "glyph" : "y"},
                        { "ascii" : "z", "glyph" : "z"},
                        { "ascii" : "A", "glyph" : "A"},
                        { "ascii" : "B", "glyph" : "B"},
                        { "ascii" : "C", "glyph" : "C"},
                        { "ascii" : "D", "glyph" : "D"},
                        { "ascii" : "E", "glyph" : "E"},
                        { "ascii" : "F", "glyph" : "F"},
                        { "ascii" : "G", "glyph" : "G"},
                        { "ascii" : "H", "glyph" : "H"},
                        { "ascii" : "I", "glyph" : "I"},
                        { "ascii" : "J", "glyph" : "J"},
                        { "ascii" : "K", "glyph" : "K"},
                        { "ascii" : "L", "glyph" : "L"},
                        { "ascii" : "M", "glyph" : "M"},
                        { "ascii" : "N", "glyph" : "N"},
                        { "ascii" : "O", "glyph" : "O"},
                        { "ascii" : "P", "glyph" : "P"},
                        { "ascii" : "Q", "glyph" : "Q"},
                        { "ascii" : "R", "glyph" : "R"},
                        { "ascii" : "S", "glyph" : "S"},
                        { "ascii" : "T", "glyph" : "T"},
                        { "ascii" : "U", "glyph" : "U"},
                        { "ascii" : "V", "glyph" : "V"},
                        { "ascii" : "W", "glyph" : "W"},
                        { "ascii" : "X", "glyph" : "X"},
                        { "ascii" : "Y", "glyph" : "Y"},
                        { "ascii" : "Z", "glyph" : "Z"},
                        { "ascii" : "0", "glyph" : "0"},
                        { "ascii" : "1", "glyph" : "1"},
                        { "ascii" : "2", "glyph" : "2"},
                        { "ascii" : "3", "glyph" : "3"},
                        { "ascii" : "4", "glyph" : "4"},
                        { "ascii" : "5", "glyph" : "5"},
                        { "ascii" : "6", "glyph" : "6"},
                        { "ascii" : "7", "glyph" : "7"},
                        { "ascii" : "8", "glyph" : "8"},
                        { "ascii" : "9", "glyph" : "9"},
                        { "ascii" : " ", "glyph" : " "},
                        { "ascii" : "%", "glyph" : "%"},
                        { "ascii" : "-", "glyph" : "-"},
                        { "ascii" : "+", "glyph" : "+"},
                        { "ascii" : ".", "glyph" : "."},
                        { "ascii" : "°", "glyph" : "°"}
                    ]
                }
            ]
        }
    ]
}
```

