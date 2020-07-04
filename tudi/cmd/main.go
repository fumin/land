package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pkg/errors"

	"github.com/fumin/land/tudi"
)

func run() error {
	fname := "example.pdf"
	password := ""

	f, err := os.Open(fname)
	if err != nil {
		return errors.Wrap(err, "")
	}
	defer f.Close()

	dihao, err := tudi.Parse(f, password)
	if err != nil {
		return errors.Wrap(err, "")
	}

	for _, dh := range dihao {
		txs := make(map[string]tudi.TaXiang)
		for _, tx := range dh.TaXiang {
			txs[tx.CiXu] = tx
		}

		for _, syq := range dh.SuoYouQuan {
			tx := txs[syq.TaXiang]

			jh := ""
			if len(tx.JianHao) > 0 {
				jh = tx.JianHao[0]
			}

			fmt.Printf("%s,%s,%s,%d/%d,%s,%s %s\n", dh.Name, syq.Owner, syq.IDNum, syq.FanWei[1], syq.FanWei[0], jh, tx.QuanLi, tx.Reason)
		}
		//fmt.Printf("%+v\n", dh)
	}
	return nil
}

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	if err := run(); err != nil {
		log.Fatalf("%+v", err)
	}
}
