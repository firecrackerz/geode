is color

include "std:math"

class Color {
	float r;
	float g;
	float b;
}

func NewColorRGB(float r, float g, float b) Color {
	Color col;
	col.r <- r;
	col.g <- g;
	col.b <- b;	
	return col;
}
				
func NewColorHSV(float h, float s, float v) Color {
	h <- h % 360.0;
	if s = 0 {
		return NewColorRGB(v, v, v);
	}
	int i := (h * 6);
	float C := v * s;
	float X := C * (1 - math:fabs((h / 60.0) % 2));
	float m := v - C;
	
	Color col;
	if   0 <= h && h < 60  {col <- NewColorRGB(C, X, 0);}
	if  60 <= h && h < 120 {col <- NewColorRGB(X, C, 0);}
	if 120 <= h && h < 180 {col <- NewColorRGB(0, C, X);}
	if 180 <= h && h < 240 {col <- NewColorRGB(0, X, C);}
	if 240 <= h && h < 300 {col <- NewColorRGB(X, 0, C);}
	if 300 <= h && h < 360 {col <- NewColorRGB(C, 0, X);}

	
	col.r <- (col.r + m) * 255;
	col.g <- (col.g + m) * 255;
	col.b <- (col.b + m) * 255;
	return col;
}