## TGA/PNG to Sprite-Pack

A golang tool to convert TGA/PNG image files to a custom binary file format that can be loaded by the library as an array of images/sprites.

Tool takes a .json file as input that describes the images to be included in the sprite pack and outputs a .bin file that contains the sprite data in the custom binary file format.

```json
{
  "files": [
    {
      "file": "sprite_map_icons.png",
      "sprites": [
         {
           "name": "sprite1",
           "format": "RGB565A1"
         }
      ]
    },
    {
      "file": "sprite_map_buttons.tga",
      "sprites": [
         {
           "name":"title",
           "format": "I8A1",
           "rect": {
               "x": 0,
               "y": 0,
               "w": 64,
               "h": 64
            }
         },
         {
           "name":"button1",
           "format": "RGBA8888",
           "rect": {
               "x": 64,
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
- u32 reserved (for alignment)
- sprite array[]
    - u16 width
    - u16 height
    - u16 format (e.g. RGBA565, RGBA565A1, RGBA8888, I8, I8A1, etc.)
    - u16 reserved (for alignment)
    - u32 pixel data size
    - u32 alpha data size
    - u64 pixel data offset in file
    - u64 alpha data offset in file (for formats with separate alpha data)
    - u64 color palette offset in file (for paletted formats)
- Data
    - data
- Color Palette Data
    - raw color palette data for each image (for paletted formats)

