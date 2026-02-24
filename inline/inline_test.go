package inline

import (
	"bytes"
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWriteJsonResponse(t *testing.T) {
	Convey("writeJsonResponse", t, func() {
		Convey("Should produce valid JSON for empty anime list", func() {
			var buf bytes.Buffer
			opts := &Options{Query: "test", Json: true}
			err := writeJson(&buf, nil, opts)
			So(err, ShouldBeNil)

			var output Output
			err = json.Unmarshal(buf.Bytes(), &output)
			So(err, ShouldBeNil)
			So(output.Query, ShouldEqual, "test")
			So(output.Result, ShouldHaveLength, 0)
		})
	})
}
