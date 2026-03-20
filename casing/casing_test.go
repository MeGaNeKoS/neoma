package casing_test

import (
	"strings"
	"testing"

	"github.com/MeGaNeKoS/neoma/casing"
	"github.com/stretchr/testify/assert"
)

func TestSplit(tt *testing.T) {
	tests := []struct {
		Input    string
		Expected []string
	}{
		{"CamelCaseTest", []string{"Camel", "Case", "Test"}},
		{"lowerCamelTest", []string{"lower", "Camel", "Test"}},
		{"snake_case_test", []string{"snake", "case", "test"}},
		{"kabob-case-test", []string{"kabob", "case", "test"}},
		{"Space delimited test", []string{"Space", "delimited", "test"}},

		{"AnyKind of_string", []string{"Any", "Kind", "of", "string"}},
		{"hello__man how-Are you??", []string{"hello", "man", "how", "Are", "you"}},
		{"UserID", []string{"User", "ID"}},
		{"HTTPServer", []string{"HTTP", "Server"}},
		{"Test123Test", []string{"Test", "123", "Test"}},
		{"Test123test", []string{"Test", "123", "test"}},
		{"Dupe-_---test", []string{"Dupe", "test"}},
		{"ÜberWürsteÄußerst", []string{"Über", "Würste", "Äußerst"}},
		{"MakeAWish", []string{"Make", "A", "Wish"}},
		{"uHTTP123", []string{"u", "HTTP", "123"}},
		{"aB1-1Ba", []string{"a", "B", "1", "1", "Ba"}},
		{"a.bc.d", []string{"a", "bc", "d"}},
		{"Emojis 🎉🎊-🎈", []string{"Emojis", "🎉🎊", "🎈"}},
		{"a b c", []string{"a", "b", "c"}},
		{"1 2 3", []string{"1", "2", "3"}},
	}

	for _, test := range tests {
		tt.Run(test.Input, func(t *testing.T) {
			assert.Equal(t, test.Expected, casing.Split(test.Input))
		})
	}
}

func TestCamelCases(t *testing.T) {
	assert.Equal(t, "CamelCaseTEST", casing.Camel("camel_case_TEST", casing.Identity))
	assert.Equal(t, "CamelCaseTest", casing.Camel("camel_case_TEST"))

	assert.Equal(t, "lowerCamelCaseTEST", casing.LowerCamel("lower_camel_case_TEST", casing.Identity))
	assert.Equal(t, "lowerCamelCaseTest", casing.LowerCamel("lower_camel_case_TEST"))

	assert.Equal(t, "überStraße", casing.LowerCamel("ÜberStraße"))
}

func TestSnakeCase(t *testing.T) {
	assert.Equal(t, "Snake_Case_TEST", casing.Snake("SnakeCaseTEST", casing.Identity))
	assert.Equal(t, "snake_case_test", casing.Snake("SnakeCaseTEST"))
	assert.Equal(t, "unsinn_überall", casing.Snake("UnsinnÜberall"))

	assert.Equal(t, "mp4", casing.Snake("mp4"))
	assert.Equal(t, "h264_stream", casing.Snake("h.264 stream"))
	assert.Equal(t, "foo1_23", casing.Snake("Foo1-23"))
	assert.Equal(t, "1stop", casing.Snake("1 stop"))
}

func TestKebabCase(t *testing.T) {
	assert.Equal(t, "Kebab-Case-TEST", casing.Kebab("KebabCaseTEST", casing.Identity))
	assert.Equal(t, "kebab-case-test", casing.Kebab("KebabCaseTEST"))
}

func TestInitialism(t *testing.T) {
	assert.Equal(t, "UserID", casing.Camel("USER_ID", strings.ToLower, casing.Initialism))
	assert.Equal(t, "PlatformAPI", casing.Camel("platform-api", casing.Initialism))
}

func TestRemovePart(t *testing.T) {
	assert.Equal(t, "one_two", casing.Snake("one-and-two-and", func(part string) string {
		if part == "and" {
			return ""
		}

		return part
	}))
}

func TestRightAlign(t *testing.T) {
	assert.Equal(t, "stream_1080p", casing.Snake("Stream1080P"))

	assert.Equal(t, "test_123foo", casing.Join(casing.MergeNumbers(casing.Split("test 123 foo"), "FOO"), "_"))
}
