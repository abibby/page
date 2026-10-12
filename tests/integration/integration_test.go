package integration_test

import (
	"testing"

	"github.com/abibby/page/test"
)

func TestBookCatalogue(t *testing.T) {
	kernel := test.Kernel(t)
	kernel.GetJSON("/api/book").AssertStatusOK().AssertJSONString(`[]`)
	kernel.GetJSON("/api/book/999").AssertStatus(404)
}
