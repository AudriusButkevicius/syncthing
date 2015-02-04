// Copyright (C) 2014 The Syncthing Authors.
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU General Public License along
// with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"io"
)

func newSeparatingWriter(w io.Writer, sep string) io.Writer {
	return &separatingWriter{
		writer:    w,
		separator: []byte(sep),
		firstdone: false,
	}
}

type separatingWriter struct {
	writer    io.Writer
	separator []byte
	firstdone bool
}

func (sw *separatingWriter) Write(data []byte) (int, error) {
	if sw.firstdone {
		sw.writer.Write(sw.separator)
	} else {
		sw.firstdone = true
	}
	return sw.writer.Write(data)
}
