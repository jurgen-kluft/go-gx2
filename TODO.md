# TODO

## TTF, OTF
- [https://github.com/BurntSushi/freetype-go](freetype-go)
- Or [https://github.com/speedata/gootf](gootf)

## BDF

- [https://github.com/zachomedia/go-bdf](go-bdf) 

## Images

- [https://github.com/ftrvxmtrx/tga](TGA)
- 


## Palette reuse

Palette reuse is currently very basic, we just identify exact matches and reuse those. This can be improved by identifying subsets of palettes and reusing those instead, which would reduce the number of palettes and thus the overall size of the sprite pack.
