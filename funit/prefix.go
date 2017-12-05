// Copyright 2017 Aleksey Blinov. All rights reserved.

package funit

// Base-10 suffixes.
const (
	Kilo  Measure = 1000.0
	Mega          = 1000.0 * Kilo
	Giga          = 1000.0 * Mega
	Tera          = 1000.0 * Giga
	Peta          = 1000.0 * Tera
	Exa           = 1000.0 * Peta
	Zetta         = 1000.0 * Exa
	Yotta         = 1000.0 * Zetta
	Milli         = 1.0 / 1000.0
	Micro         = Milli / 1000.0
	Nano          = Micro / 1000.0
	Pico          = Nano / 1000.0
	Femto         = Pico / 1000.0
)

// Base-2 suffixes - long and short.
const (
	Kibi Measure = 1024.0
	Mebi         = 1024.0 * Kibi
	Gibi         = 1024.0 * Mebi
	Tebi         = 1024.0 * Gibi
	Pebi         = 1024.0 * Tebi
	Exbi         = 1024.0 * Pebi
	Zebi         = 1024.0 * Exbi
	Yobi         = 1024.0 * Zebi
	Ki           = Kibi
	Mi           = Mebi
	Gi           = Gibi
	Ti           = Tebi
	Pi           = Pebi
	Ei           = Exbi
	Zi           = Zebi
	Yi           = Yobi
)
