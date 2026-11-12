package pdf

import (
	"testing"

	"github.com/giraffesyo/pdf/pdftest"
)

func TestGB1FallbackKeepsFullTextAndToUnicodePriority(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		font := `<< /Type /Font /Subtype /Type0 /Encoding /Identity-H /DescendantFonts [6 0 R] >>`
		want := "1目郎\uFE00"
		if explicit {
			font = `<< /Type /Font /Subtype /Type0 /Encoding /Identity-H /DescendantFonts [6 0 R] /ToUnicode 7 0 R >>`
			want = "XYZ"
		}
		raw := pdftest.Build(1,
			pdftest.Catalog(2), pdftest.Pages(3),
			pdftest.Page(2, 4, "<< /Font << /F1 5 0 R >> >>"),
			pdftest.Stream("", "BT /F1 12 Tf 50 750 Td <033e0b22560f> Tj ET"),
			font,
			`<< /Type /Font /Subtype /CIDFontType2 /DW 1000 /CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 6 >> >>`,
			pdftest.ToUnicodeCMap("3 beginbfchar\n<033e> <0058>\n<0b22> <0059>\n<560f> <005a>\nendbfchar"),
		)
		doc, err := extractOptions(t, raw, Options{Strict: true})
		if err != nil {
			t.Fatal(err)
		}
		if glyphText(doc.Pages[0].Glyphs) != want {
			t.Fatalf("explicit=%v: got %q, %v; want %q", explicit, glyphText(doc.Pages[0].Glyphs), err, want)
		}
	}
}
