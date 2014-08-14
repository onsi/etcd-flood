package flood

import (
	"fmt"
	"strings"
)

const defaultStyle = "\x1b[0m"
const redColor = "\x1b[91m"
const greenColor = "\x1b[32m"
const yellowColor = "\x1b[33m"
const cyanColor = "\x1b[36m"

func GreenBanner(s string) {
	banner(s, greenColor)
}

func CyanBanner(s string) {
	banner(s, cyanColor)
}

func RedBanner(s string) {
	banner(s, redColor)
}

func YellowBanner(s string) {
	banner(s, yellowColor)
}

func banner(s string, color string) {
	l := len(strings.Split(s, "\n")[0])
	fmt.Println("")
	fmt.Println(color + strings.Repeat("~", l) + defaultStyle)
	fmt.Println(color + s + defaultStyle)
	fmt.Println(color + strings.Repeat("~", l) + defaultStyle)
}
