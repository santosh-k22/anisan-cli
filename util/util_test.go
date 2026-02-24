package util

import (
	"regexp"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSanitizeFilename(t *testing.T) {
	Convey("SanitizeFilename", t, func() {
		Convey("Should replace invalid chars", func() {
			So(SanitizeFilename("file:name?.txt"), ShouldEqual, "file_name_.txt")
		})
		Convey("Should collapse underscores", func() {
			So(SanitizeFilename("file__name.txt"), ShouldEqual, "file_name.txt")
		})
		Convey("Should trim separators", func() {
			So(SanitizeFilename("-file-name-"), ShouldEqual, "file-name")
		})
	})
}

func TestQuantify(t *testing.T) {
	Convey("Quantify", t, func() {
		So(Quantify(1, "file", "files"), ShouldEqual, "1 file")
		So(Quantify(2, "file", "files"), ShouldEqual, "2 files")
	})
}

func TestCapitalize(t *testing.T) {
	Convey("Capitalize", t, func() {
		So(Capitalize("hello"), ShouldEqual, "Hello")
		So(Capitalize(""), ShouldEqual, "")
	})
}

func TestReGroups(t *testing.T) {
	Convey("ReGroups", t, func() {
		re := regexp.MustCompile(`(?P<first>\w+)\s(?P<last>\w+)`)
		groups := ReGroups(re, "John Doe")
		So(groups["first"], ShouldEqual, "John")
		So(groups["last"], ShouldEqual, "Doe")
	})
}

func TestFileStem(t *testing.T) {
	Convey("FileStem", t, func() {
		So(FileStem("path/to/file.txt"), ShouldEqual, "file")
		So(FileStem("file"), ShouldEqual, "file")
	})
}

func TestMaxMin(t *testing.T) {
	Convey("Max/Min", t, func() {
		So(Max(1, 5, 2), ShouldEqual, 5)
		So(Min(1, 5, 2), ShouldEqual, 1)
	})
}

func TestStack(t *testing.T) {
	Convey("Stack", t, func() {
		var s Stack[int]
		s.Push(1)
		s.Push(2)
		So(s.Len(), ShouldEqual, 2)
		So(s.Peek(), ShouldEqual, 2)
		item := s.Pop()
		So(item, ShouldEqual, 2)
		item = s.Pop()
		So(item, ShouldEqual, 1)
		item = s.Pop()
		So(item, ShouldEqual, 0)
	})
}
