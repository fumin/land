package tudi

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/unidoc/unipdf/extractor"
	pdf "github.com/unidoc/unipdf/model"
)

var (
	SuoYouQuanRe    = regexp.MustCompile(`\*\*\*[　\s]*[土㈯][地㆞]所[有㈲]權部[　\s]*\*\*\*`)
	TaXiangBuRe     = regexp.MustCompile(`\*\*\*[　\s]*[土㈯][地㆞]他[項㊠]權利部[　\s]*\*\*\*`)
	DengJiRe        = regexp.MustCompile(`）登記次序：([\d-]+)`)
	SuoYouQuanRenRe = regexp.MustCompile(`所[有㈲]權[人㆟]：(.+)`)

	// 權利範圍：******1000分之1*********
	FanWeiRe = regexp.MustCompile(`權利範圍：[全部]*[　\s]*\*+(\d+)分之(\d+)\*+`)

	// 相關他項權利登記次序：0001-000
	TaXiangRe = regexp.MustCompile(`相關他[項㊠]權利登記次序：([\d-]+)`)

	TongYiBianHao    = regexp.MustCompile(`統㆒編號：([A-Z][0-9*]+)`)
	QuanLiZhongLeiRe = regexp.MustCompile(`權利種類：(.+)`)
	DengJiYuanYinRe  = regexp.MustCompile(`登記原因：(.+)`)
	DanBaoJianHaoRe = regexp.MustCompile(`共同擔保建號：.+`)
	QiTaDengJiRe     = regexp.MustCompile(`其他登記事[項㊠]：`)
	JianHaoRe        = regexp.MustCompile(`\d+-\d+`)
)

type SuoYouQuan struct {
	Owner   string
	IDNum   string
	FanWei  [2]int
	TaXiang string
}

func extractSYQ(lines []string) ([]string, error) {
	start := -1
	end := len(lines)
	for i, l := range lines {
		if len(SuoYouQuanRe.FindAllStringSubmatchIndex(l, -1)) > 0 {
			start = i
		}
		if len(TaXiangBuRe.FindAllStringSubmatchIndex(l, -1)) > 0 {
			end = i
		}
	}
	if start < 0 {
		return nil, errors.Errorf("%+v", lines)
	}
	return lines[start:end], nil
}

func parseSYQs(lines []string) ([]SuoYouQuan, error) {
	splitted := make([][]string, 0)
	cur := make([]string, 0)

	for _, l := range lines {
		if len(DengJiRe.FindAllStringSubmatchIndex(l, -1)) > 0 {
			splitted = append(splitted, cur)
			cur = make([]string, 0)
		}
		cur = append(cur, l)
	}
	// Last batch.
	splitted = append(splitted, cur)
	// First batch is noise.
	splitted = splitted[1:]

	syqs := make([]SuoYouQuan, 0)
	for i, ls := range splitted {
		syq, err := parseSYQSingle(ls)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("%d %+v", i, ls))
		}
		syqs = append(syqs, syq)
	}
	return syqs, nil
}

func parseSYQSingle(lines []string) (SuoYouQuan, error) {
	syq := SuoYouQuan{}
	for _, l := range lines {
		if r := SuoYouQuanRenRe.FindAllStringSubmatchIndex(l, -1); len(r) > 0 {
			syq.Owner = l[r[0][2]:r[0][3]]
		}
		if r := FanWeiRe.FindAllStringSubmatchIndex(l, -1); len(r) > 0 {
			fw0 := l[r[0][2]:r[0][3]]
			fw1 := l[r[0][4]:r[0][5]]
			var err error
			syq.FanWei[0], err = strconv.Atoi(fw0)
			if err != nil {
				return SuoYouQuan{}, errors.Wrap(err, fmt.Sprintf("%s", l))
			}
			syq.FanWei[1], err = strconv.Atoi(fw1)
			if err != nil {
				return SuoYouQuan{}, errors.Wrap(err, fmt.Sprintf("%s", l))
			}
		}
		if r := TongYiBianHao.FindAllStringSubmatchIndex(l, -1); len(r) > 0 {
			syq.IDNum = l[r[0][2]:r[0][3]]
		}
		if r := TaXiangRe.FindAllStringSubmatchIndex(l, -1); len(r) > 0 {
			syq.TaXiang = l[r[0][2]:r[0][3]]
		}
	}
	if syq.Owner == "" {
		return SuoYouQuan{}, errors.Errorf("%+v", lines)
	}
	return syq, nil
}

type TaXiang struct {
	CiXu    string
	QuanLi  string
	Reason  string
	JianHao []string
}

func extractTX(lines []string) ([]string, error) {
	start := len(lines)
	for i, l := range lines {
		if len(TaXiangBuRe.FindAllStringSubmatchIndex(l, -1)) > 0 {
			start = i
		}
	}
	return lines[start:], nil
}

func parseTXSingle(lines []string) (TaXiang, error) {
	tx := TaXiang{}
	jianHaoIdx := -1
	qitaIdx := -1
	for i, l := range lines {
		if r := DengJiRe.FindAllStringSubmatchIndex(l, -1); len(r) > 0 {
			tx.CiXu = l[r[0][2]:r[0][3]]
		}
		if r := QuanLiZhongLeiRe.FindAllStringSubmatchIndex(l, -1); len(r) > 0 {
			tx.QuanLi = l[r[0][2]:r[0][3]]
		}
		if r := DengJiYuanYinRe.FindAllStringSubmatchIndex(l, -1); len(r) > 0 {
			tx.Reason = l[r[0][2]:r[0][3]]
		}
		if r := DanBaoJianHaoRe.FindAllStringSubmatchIndex(l, -1); len(r) > 0 {
			jianHaoIdx = i
		}
		if r := QiTaDengJiRe.FindAllStringSubmatchIndex(l, -1); len(r) > 0 {
			qitaIdx = i
		}
	}

	if jianHaoIdx >= 0 && qitaIdx >= 0 {
		var err error
		tx.JianHao, err = parseJianHao(lines[jianHaoIdx:qitaIdx])
		if err != nil {
			return TaXiang{}, errors.Wrap(err, "")
		}
	}

	if tx.CiXu == "" {
		return TaXiang{}, errors.Errorf("%+v", lines)
	}
	return tx, nil
}

func parseTXs(lines []string) ([]TaXiang, error) {
	if len(lines) == 0 {
		return []TaXiang{}, nil
	}

	splitted := make([][]string, 0)
	cur := make([]string, 0)

	for _, l := range lines {
		if len(DengJiRe.FindAllStringSubmatchIndex(l, -1)) > 0 {
			splitted = append(splitted, cur)
			cur = make([]string, 0)
		}
		cur = append(cur, l)
	}
	// Last batch.
	splitted = append(splitted, cur)
	// First batch is noise.
	splitted = splitted[1:]

	txs := make([]TaXiang, 0)
	for i, ls := range splitted {
		tx, err := parseTXSingle(ls)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("%d %+v", i, ls))
		}
		txs = append(txs, tx)
	}
	return txs, nil

}

func parseJianHao(lines []string) ([]string, error) {
	joined := strings.Join(lines, "\n")
	r := JianHaoRe.FindAllStringSubmatchIndex(joined, -1)
	jh := make([]string, 0)
	for _, ri := range r {
		jh = append(jh, joined[ri[0]:ri[1]])
	}
	return jh, nil
}

type DiHao struct {
	Name       string
	SuoYouQuan []SuoYouQuan
	TaXiang    []TaXiang
}

type Parser struct {
	Cur      string
	CurLines []string
	DiHao    []DiHao
}

func parseDihao(lines []string) (DiHao, error) {
	dh := DiHao{}
	dh.Name = lines[1]

	syqLines, err := extractSYQ(lines)
	if err != nil {
		return DiHao{}, errors.Wrap(err, "")
	}
	dh.SuoYouQuan, err = parseSYQs(syqLines)
	if err != nil {
		return DiHao{}, errors.Wrap(err, "")
	}

	txLines, err := extractTX(lines)
	if err != nil {
		return DiHao{}, errors.Wrap(err, "")
	}
	dh.TaXiang, err = parseTXs(txLines)
	if err != nil {
		return DiHao{}, errors.Wrap(err, "")
	}

	return dh, nil
}

func (p *Parser) Parse(page []string) error {
	dihaoName, err := parseHeader(page)
	if err != nil {
		return errors.Wrap(err, "")
	}
	if dihaoName != p.Cur && p.Cur != "" {
		if err := p.ParseCur(); err != nil {
			return errors.Wrap(err, "")
		}
	}
	p.Cur = dihaoName
	p.CurLines = append(p.CurLines, page...)
	return nil
}

func (p *Parser) ParseCur() error {
	dihao, err := parseDihao(p.CurLines)
	if err != nil {
		return errors.Wrap(err, "")
	}
	p.DiHao = append(p.DiHao, dihao)
	p.CurLines = make([]string, 0)
	return nil
}

func parseHeader(page []string) (string, error) {
	if strings.Contains(page[0], "謄本") {
		return page[1], nil
	}
	return page[0], nil
}

func trimLines(lines []string) []string {
	res := make([]string, 0, len(lines))
	for _, l := range lines {
		res = append(res, strings.TrimSpace(l))
	}
	return res
}

func Parse(reader io.ReadSeeker, password string) ([]DiHao, error) {
	pdfReader, err := pdf.NewPdfReader(reader)
	if err != nil {
		log.Printf("+%v", err)
		return nil, errors.Wrap(err, "")
	}
	isEncrypted, err := pdfReader.IsEncrypted()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	if isEncrypted {
		auth, err := pdfReader.Decrypt([]byte(password))
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		if !auth {
			return nil, fmt.Errorf("Wrong password")
		}
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	parser := Parser{}
	for i := 0; i < numPages; i++ {
		pageNum := i + 1
		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}

		ex, err := extractor.New(page)
		if err != nil {
			return nil, errors.Wrap(err, "")
		}
		text, err := ex.ExtractText()
		if err != nil {
			return nil, errors.Wrap(err, "")
		}

		lines := strings.Split(text, "\n")
		lines = trimLines(lines)
		if err := parser.Parse(lines); err != nil {
			return nil, errors.Wrap(err, "")
		}
	}
	if err := parser.ParseCur(); err != nil {
		return nil, errors.Wrap(err, "")
	}

	return parser.DiHao, nil
}
