# Name = "Overload"
# ExpectedOutput = "2 2.0"
is main

include "std::io"

func foo(int a) int {
	return a * 2;
}

func foo(float a) float {
	return a * 2;
}

func main() int {
	io:print("%d %.1f", foo(1), foo(1.0));
	return 0;
}