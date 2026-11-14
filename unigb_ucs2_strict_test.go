package pdf

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/giraffesyo/pdf/pdftest"
)

func strictUCS2Fixture(encoding, codes, operator string) []byte {
	body := "BT /F1 10 Tf 72 700 Td <" + codes + "> Tj ET"
	switch operator {
	case "TJ":
		body = "BT /F1 10 Tf 72 700 Td [<" + codes + ">] TJ ET"
	case "quote":
		body = "BT /F1 10 Tf 72 700 Td <" + codes + "> ' ET"
	case "double_quote":
		body = "BT /F1 10 Tf 72 700 Td 0 0 <" + codes + "> \" ET"
	case "split":
		body = "BT /F1 10 Tf 72 700 Td [<4E> <2D>] TJ ET"
	}
	resources := "<< /Font << /F1 5 0 R >> /XObject << /Fm 7 0 R >> >>"
	page := pdftest.Page(2, 4, resources)
	content := body
	extras := []string{}
	switch operator {
	case "form":
		content = "/Fm Do"
		extras = append(extras, pdftest.Stream("/Type /XObject /Subtype /Form /BBox [0 0 300 800] /Resources << /Font << /F1 5 0 R >> >>", body))
	case "annotation":
		content = ""
		page = strings.TrimSuffix(page, ">>") + " /Annots [7 0 R] >>"
		extras = append(extras, "<< /Type /Annot /Subtype /FreeText /Rect [0 0 300 800] /AP << /N 8 0 R >> >>", pdftest.Stream("/Type /XObject /Subtype /Form /BBox [0 0 300 800] /Resources << /Font << /F1 5 0 R >> >>", body))
	}
	objects := []string{pdftest.Catalog(2), pdftest.Pages(3), page, pdftest.Stream("", content), "<< /Type /Font /Subtype /Type0 /BaseFont /Synthetic /Encoding /" + encoding + " /DescendantFonts [6 0 R] >>", "<< /Type /Font /Subtype /CIDFontType2 /BaseFont /Synthetic /DW 1000 /CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 2 >> >>"}
	return pdftest.Build(1, append(objects, extras...)...)
}

func strictBuiltinResolver(string) ([]byte, error) { return nil, ErrUseBuiltinCMap }

func TestStrictBuiltinUniGBUCS2AllTextEntrypoints(t *testing.T) {
	for _, operator := range []string{"Tj", "TJ", "quote", "double_quote", "form", "annotation"} {
		t.Run(operator, func(t *testing.T) {
			for _, tc := range []struct {
				name, codes string
				invalid     bool
			}{{"valid", "4E2D658700350025", false}, {"odd_ascii_tail", "4E2D41", true}, {"odd_zero_tail", "4E2D00", true}, {"high_surrogate", "4E2DD800", true}, {"low_surrogate", "4E2DDFFF", true}, {"UTF16_pair_is_not_UCS2", "4E2DD83DDE00", true}} {
				t.Run(tc.name, func(t *testing.T) {
					raw := strictUCS2Fixture("UniGB-UCS2-H", tc.codes, operator)
					doc, e := ExtractWithOptions(context.Background(), bytes.NewReader(raw), int64(len(raw)), Options{Strict: true, CMapResolver: strictBuiltinResolver})
					if !tc.invalid {
						if e != nil || doc.Text() != "中文5%" {
							t.Fatalf("strict Unicode lost actual text: doc=%v err=%v", doc, e)
						}
						return
					}
					var strict *StrictError
					if !errors.As(e, &strict) || strict.Warning.Code != WarningMalformedPage {
						t.Fatalf("invalid raw UCS2 was not a strict malformed warning: %v", e)
					}
				})
			}
		})
	}
	raw := strictUCS2Fixture("UniGB-UCS2-H", "", "split")
	_, e := ExtractWithOptions(context.Background(), bytes.NewReader(raw), int64(len(raw)), Options{Strict: true, CMapResolver: strictBuiltinResolver})
	var strict *StrictError
	if !errors.As(e, &strict) || strict.Warning.Code != WarningMalformedPage {
		t.Fatal("UCS2 code crossed TJ string boundaries", e)
	}
}

func TestStrictBuiltinUniGBIsExplicitAndDoesNotGuessCIDNames(t *testing.T) {
	for _, name := range []string{"Unknown-CMap", "UniUnknown-UCS2-H", "UniGB-UCS2-V", "UniGB-UTF16-H", "UniGB-UCS2-H#20"} {
		raw := strictUCS2Fixture(name, "4E2D6587", "Tj")
		if _, e := ExtractWithOptions(context.Background(), bytes.NewReader(raw), int64(len(raw)), Options{Strict: true, CMapResolver: strictBuiltinResolver}); e == nil {
			t.Fatal("unknown name used strict Unicode", name)
		}
	}
	for _, name := range []string{"Identity-H", "Identity-V"} {
		raw := strictUCS2Fixture(name, "4E2D6587", "Tj")
		doc, e := ExtractWithOptions(context.Background(), bytes.NewReader(raw), int64(len(raw)), Options{Strict: true, CMapResolver: strictBuiltinResolver})
		if e != nil || strings.Contains(doc.Text(), "中文") {
			t.Fatal("CID Identity was treated as Unicode", name, e)
		}
	}
	raw := strictUCS2Fixture("UniGB-UCS2-H", "4E2D41", "Tj")
	doc, e := ExtractWithOptions(context.Background(), bytes.NewReader(raw), int64(len(raw)), Options{Strict: true})
	if e != nil || doc.Text() != "中A" {
		t.Fatal("legacy nil-resolver behavior changed", e)
	}
}

func TestStrictBuiltinUniGBDoesNotLoseValidationThroughInheritance(t *testing.T) {
	raw := pdftest.Build(1, pdftest.Catalog(2), pdftest.Pages(3), pdftest.Page(2, 4, "<< /Font << /F1 5 0 R >> >>"), pdftest.Stream("", "BT /F1 10 Tf 72 700 Td <4E2D6587> Tj ET"), "<< /Type /Font /Subtype /Type0 /BaseFont /Synthetic /Encoding 7 0 R /DescendantFonts [6 0 R] >>", "<< /Type /Font /Subtype /CIDFontType2 /DW 1000 >>", pdftest.Stream("", "/UniGB-UCS2-H usecmap"))
	if _, e := ExtractWithOptions(context.Background(), bytes.NewReader(raw), int64(len(raw)), Options{Strict: true, CMapResolver: strictBuiltinResolver}); e == nil {
		t.Fatal("inherited CMap escaped strict named Encoding boundary")
	}
	raw = strictUCS2Fixture("UniGB-UCS2-H", "4E2D41", "Tj")
	doc, e := ExtractWithOptions(context.Background(), bytes.NewReader(raw), int64(len(raw)), Options{CMapResolver: strictBuiltinResolver})
	if e != nil || len(doc.Warnings) != 1 || doc.Warnings[0].Code != WarningMalformedPage {
		t.Fatal("permissive extraction silently lost malformed UCS2", e)
	}
}
