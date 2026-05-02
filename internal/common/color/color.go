package color

import fcolor "github.com/fatih/color"

var Success = fcolor.New(fcolor.FgGreen).SprintFunc()
var Warn = fcolor.New(fcolor.FgYellow).SprintFunc()
var Error = fcolor.New(fcolor.FgRed).SprintFunc()
