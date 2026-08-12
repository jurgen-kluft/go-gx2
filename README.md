# go-gx2

go-gx2 is a Go toolkit for building compact binary asset packs for embedded or resource-constrained UI runtimes.

The repository currently focuses on two pipelines:
- Font pack generation from TTF/OTF/BDF sources.
- Sprite pack generation from PNG/TGA source images.

It also includes reusable readers/writers for those binary pack formats and a maintained TGA codec package.

## Why this repository exists

Many UI stacks need very fast startup and predictable memory use. This project converts source assets into binary structures that can be memory-mapped or loaded directly by native code (for example C/C++ runtimes), minimizing runtime parsing overhead.

## Repository layout

- [fontpak](fontpak): Font config parsing, font building, and binary read/write.
- [spritepak](spritepak): Sprite config model, image conversion pipeline, and binary read/write.
- [cmd/fontpak](cmd/fontpak): CLI entrypoint for font pack generation.
- [cmd/spritepak](cmd/spritepak): CLI entrypoint for sprite pack generation.
- [bdf](bdf): BDF parser used by the font pipeline.
- [tga](tga): TGA encoder/decoder package.
- [docs/fontpack.md](docs/fontpack.md): Extended notes for font pack format and config.
- [docs/spritepack.md](docs/spritepack.md): Extended notes for sprite pack format and config.

## Core capabilities

### Font pipeline

The font pipeline can:
- Load a JSON config with one or more font files and named output fonts.
- Build glyph bitmaps and metrics for an explicit ASCII-to-glyph map.
- Support TTF/OTF rendering and BDF ingestion.
- Write/read a binary font pack structure.

Primary API surface in [fontpak](fontpak):
- LoadConfig(path string) (error, *Config)
- BuildFontPak(cfg *Config) (error, *FontPack)
- WritePack(w io.Writer, fontPack *FontPack) error
- ReadPack(r io.Reader) (*FontPack, error)

### Sprite pipeline

The sprite pipeline can:
- Load PNG or TGA source images.
- Slice sprites from full images or explicit rectangles.
- Convert to compact pixel formats.
- Store optional separate alpha streams and optional indexed palettes.
- Write/read a binary sprite pack structure.

Primary API surface in [spritepak](spritepak):
- Build(cfg Configuration) ([]SpriteEntry, error)
- WritePack(w io.Writer, sprites []SpriteEntry) error
- ReadPack(r io.Reader) (*SpritePack, error)

## Supported sprite formats

Configured format strings currently accepted by the builder:
- RGB565
- RGB565A1
- RGB565A4 (accepted, currently falls back to RGBA8888)
- RGB565A8 (accepted, currently falls back to RGBA8888)
- RGBA8888
- I8 (indexed 8-bit palette)

## Command-line tools

Build the tools:

~~~bash
go build ./cmd/fontpak
go build ./cmd/spritepak
~~~

Run font pack tool:

~~~bash
go run ./cmd/fontpak <config.json> <out.bin>
~~~

Run sprite pack tool:

~~~bash
go run ./cmd/spritepak <config.json> <out.bin>
~~~

## Font config shape (summary)

The font config is a JSON document with top-level files[] where each file entry contains one or more named fonts and character mappings.

Minimal example:

~~~json
{
    "mapping": {
        "FiraCodeNerdFontMono-Regular": 0
    },
    "fonts": [
        {
            "file": "/Users/obnosis5/Library/Fonts/FiraCodeNerdFontMono-Regular.ttf",
            "name": "FiraCodeNerdFontMono-Regular",
            "options" : {
                "size": 24,
                "sdf": true,
                "sdf_border": 2,
                "sdf_radius": 1.0,
                "sdf_cutoff": 0.05
            }
        }
    ],
    "chars": [
        { "address": "a", "glyph": "a" },
        { "address": "b", "glyph": "b" }
    ]
}
~~~

See [docs/fontpack.md](docs/fontpack.md) for a larger example and C/C++ interop notes.

## Sprite config shape (summary)

The sprite config is a JSON document with files[] entries. Each entry selects an image file and one or more sprite descriptors.

Minimal example:

~~~json
{
    "sprites": [
        {
            "image_file": "WeatherSprites.png",
            "sprites_file": "WeatherSprites.json",
            "palette_name": "WeatherSprites"
        }
    ],
    "sprites_mapping": {
        "celcius": 0,
        "clouds.fog": 1,
        "clouds.hail": 2
    }
}
~~~

See [docs/spritepack.md](docs/spritepack.md) for additional details.

## Using as a library

### Build and write a font pack

~~~go
cfgErr, cfg := fontpack.LoadConfig("fontpack.json")
if cfgErr != nil {
		panic(cfgErr)
}

buildErr, pack := fontpack.BuildFontPak(cfg)
if buildErr != nil {
		panic(buildErr)
}

f, err := os.Create("fontpack.bin")
if err != nil {
		panic(err)
}
defer f.Close()

if err := fontpack.WritePack(f, pack); err != nil {
		panic(err)
}
~~~

### Build and write a sprite pack

~~~go
sprites, err := spritepack.Build(cfg)
if err != nil {
		panic(err)
}

f, err := os.Create("spritepack.bin")
if err != nil {
		panic(err)
}
defer f.Close()

if err := spritepack.WritePack(f, sprites); err != nil {
		panic(err)
}
~~~

## Development

Run tests:

~~~bash
go test ./...
~~~

Current tests are concentrated in:
- [spritepak/read_write_test.go](spritepak/read_write_test.go)
- [tga/decode_test.go](tga/decode_test.go)
- [tga/encode_test.go](tga/encode_test.go)

## Dependencies

Declared in [go.mod](go.mod):
- golang.org/x/image
- golang.org/x/text

The pack read/write path also uses the codestream package from github.com/jurgen-kluft/go-datastream.

## Status and roadmap

Ideas and pending improvements are tracked in [TODO.md](TODO.md), including work around font source options and improved palette reuse.

