package pdf

import (
	"bytes"
	"context"
	"testing"

	"github.com/giraffesyo/pdf/pdftest"
)

func renderModePDF(body string) []byte {
	return pdftest.Build(1, pdftest.Catalog(2), pdftest.Pages(3),
		pdftest.Page(2, 4, "<< /Font << /F1 5 0 R >> /XObject << /Fm 6 0 R >> >>"),
		pdftest.Stream("", body), pdftest.Helvetica(),
		pdftest.Stream("/Type /XObject /Subtype /Form /BBox [0 0 100 100]", "BT 3 Tr /F1 12 Tf 0 0 Td (form) Tj ET"))
}

func TestPageTextRenderModesFollowGraphicsState(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       uint8
	}{
		{"default", "BT /F1 12 Tf (visible) Tj ET", 1},
		{"persistent_BT_ET", "BT 3 Tr ET BT /F1 12 Tf (hidden) Tj ET", 8},
		{"q_Q_restore", "q BT 3 Tr /F1 12 Tf (hidden) Tj ET Q BT /F1 12 Tf (visible) Tj ET", 9},
		{"form_shared_summary_local_state", "/Fm Do BT /F1 12 Tf (visible) Tj ET", 9},
		{"empty", "BT 3 Tr /F1 12 Tf () Tj ET", 0},
		{"clip_only", "BT 7 Tr /F1 12 Tf (clip) Tj ET", 128},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := renderModePDF(tc.body)
			doc, err := ExtractWithOptions(context.Background(), bytes.NewReader(raw), int64(len(raw)), Options{Strict: true})
			if err != nil {
				t.Fatal(err)
			}
			if got := doc.Pages[0].TextRenderModes; got != tc.want {
				t.Fatalf("text rendering modes=%d; want %d", got, tc.want)
			}
		})
	}
}

func TestPageTextRenderModesDoNotLeakAcrossPages(t *testing.T) {
	resources := "<< /Font << /F1 7 0 R >> >>"
	raw := pdftest.Build(1, pdftest.Catalog(2), pdftest.Pages(3, 4),
		pdftest.Page(2, 5, resources), pdftest.Page(2, 6, resources),
		pdftest.Stream("", "BT 3 Tr /F1 12 Tf (hidden) Tj ET"),
		pdftest.Stream("", "BT /F1 12 Tf (visible) Tj ET"), pdftest.Helvetica())
	doc, err := ExtractWithOptions(context.Background(), bytes.NewReader(raw), int64(len(raw)), Options{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Pages[0].TextRenderModes != 8 || doc.Pages[1].TextRenderModes != 1 {
		t.Fatal("rendering mode state leaked across pages")
	}
}

func TestInvalidTextRenderModeFailsStrict(t *testing.T) {
	for _, value := range []string{"8", "-1", "1.5", "()", "/Unknown", ""} {
		raw := renderModePDF("BT " + value + " Tr /F1 12 Tf (untrusted) Tj ET")
		if _, err := ExtractWithOptions(context.Background(), bytes.NewReader(raw), int64(len(raw)), Options{Strict: true}); err == nil {
			t.Fatalf("invalid Tr %q accepted", value)
		}
	}
}
