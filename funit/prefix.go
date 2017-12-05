// Copyright 2017 Aleksey Blinov. All rights reserved.

package funit

// Base-10 suffixes.
const (
	Kilo  Measure = 1000
	Mega          = 1000 * Kilo
	Giga          = 1000 * Mega
	Tera          = 1000 * Giga
	Peta          = 1000 * Tera
	Exa           = 1000 * Peta
	Zetta         = 1000 * Exa
	Yotta         = 1000 * Zetta
	Milli         = 1 / 1000
	Micro         = Milli / 1000
	Nano          = Micro / 1000
	Pico          = Nano / 1000
	Femto         = Pico / 1000
)

// Base-2 suffixes - long and short.
const (
	Kibi Measure = 1024
	Mebi         = 1024 * Kibi
	Gibi         = 1024 * Mebi
	Tebi         = 1024 * Gibi
	Pebi         = 1024 * Tebi
	Exbi         = 1024 * Pebi
	Zebi         = 1024 * Exbi
	Yobi         = 1024 * Zebi
	Ki           = Kibi
	Mi           = Mebi
	Gi           = Gibi
	Ti           = Tebi
	Pi           = Pebi
	Ei           = Exbi
	Zi           = Zebi
	Yi           = Yobi
)
