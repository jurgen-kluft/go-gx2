package fx_rotozoom

import fx_common "github.com/jurgen-kluft/go-gx2/test-apps/effects/common"

// static const uint8_t PIXEL_SIZE = 2;
// static const uint8_t SPEED = 2;

// static uint16_t angle;
// // static float sinlut[360];
// // static float coslut[360];

// void
// rotozoom_init()
// {
//     /* Generate look up tables. */
//     // for (uint16_t i = 0; i < 360; i++) {
//     //     sinlut[i] = sin(i * M_PI / 180);
//     //     coslut[i] = cos(i * M_PI / 180);
//     // }
// }

// void
// rotozoom_render(hagl_backend_t const *display)
// {
//     float s, c, z;

//     s = sin(angle * M_PI / 180);
//     c = cos(angle * M_PI / 180);
//     // s = sinlut[angle];
//     // c = coslut[angle];
//     z = s * 1.2;

//     for (uint16_t x = 0; x < DISPLAY_WIDTH; x = x + PIXEL_SIZE) {
//         for (uint16_t y = 0; y < DISPLAY_HEIGHT; y = y + PIXEL_SIZE) {

//             /* Get a rotated pixel from the head image. */
//             int16_t u = (int16_t)((x * c - y * s) * z) % HEAD_WIDTH;
//             int16_t v = (int16_t)((x * s + y * c) * z) % HEAD_HEIGHT;

//             u = abs(u);
//             if (v < 0) {
//                 v += HEAD_HEIGHT;
//             }
//             hagl_color_t *color = (hagl_color_t *) (head + HEAD_WIDTH * sizeof(hagl_color_t) * v + sizeof(hagl_color_t) * u);

//             if (1 == PIXEL_SIZE) {
//                 hagl_put_pixel(display, x, y, *color);
//             } else {
//                 hagl_fill_rectangle(display, x, y, x + PIXEL_SIZE - 1, y + PIXEL_SIZE - 1, *color);
//             }
//             // hagl_put_pixel(x, y, *color);
//         }
//     }
// }

// void
// rotozoom_animate()
// {
//     angle = (angle + SPEED) % 360;
// }

type Effect struct {
	Angle int32
	Speed int32

	ImageData   []uint8
	ImageWidth  int32
	ImageHeight int32
}

func NewEffect(speed int32) *Effect {
	f := &Effect{
		Angle: 0,
		Speed: speed,
	}

	f.ImageData = IMAGE_DATA
	f.ImageWidth = IMAGE_WIDTH
	f.ImageHeight = IMAGE_HEIGHT

	return f
}

func (e *Effect) Animate() {
	e.Angle = (e.Angle + e.Speed)
	if e.Angle >= 360 {
		e.Angle -= 360
	}
}

func (e *Effect) Render(fb *fx_common.FrameBuffer) {
}
