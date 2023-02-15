// Copyright 2018 The gttc Authors
// This file is part of the gttc library.
//
// The gttc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gttc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gttc library. If not, see <http://www.gnu.org/licenses/>.

// PM want to change the prefix of hex from 0x to anything they want, @#$%@#$%@#%^ ;-)
// Just set the CustomHashPrefix to "0x" , everything will back to normal.

package hexutil

const CustomHexPrefix = "0x"

var PossibleCustomHexPrefixMap = map[string]bool{
	"0x": true,
	"0X": true,
	"t0": true,
	"t1": true,
	"t2": true,
	"t3": true,
	"t4": true,
	"t5": true,
	"t6": true,
	"t7": true,
	"t8": true,
	"t9": true,
	"T0": true,
	"T1": true,
	"T2": true,
	"T3": true,
	"T4": true,
	"T5": true,
	"T6": true,
	"T7": true,
	"T8": true,
	"T9": true,
}

func CPToHex(s string) string {
	if len(s) > len(CustomHexPrefix) {
		if _, ok := PossibleCustomHexPrefixMap[s[:2]]; ok {
			return "0x" + s[2:]
		}
	}
	return s
}

func HexToCP(s string) string {
	if len(s) > 2 {
		if s[:2] == "0x" || s[:2] == "0X" {
			return CustomHexPrefix + s[2:]
		}
	}
	return s
}
