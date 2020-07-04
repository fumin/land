package tudi

import (
	"flag"
	"log"
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()
	fname := "example.pdf"
	password := ""

	f, err := os.Open(fname)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer f.Close()
	dihao, err := Parse(f, password)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if len(dihao) != 3 {
		t.Fatalf("%+v", dihao)
	}
	// log.Printf("%+v", dihao)
}

func TestMain(m *testing.M) {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	os.Exit(m.Run())
}
