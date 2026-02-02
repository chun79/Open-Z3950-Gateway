package z3950

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"regexp"
)

var isbnCleanRegex = regexp.MustCompile(`[^0-9xX]`)

type MARCField struct {
	Tag   string
	Value string
}

type Holding struct {
	CallNumber string `json:"call_number"`
	Status     string `json:"status"`   // "Available", "Checked Out", "Lost"
	Location   string `json:"location"` // "Main Library", "Science Branch"
}

type MARCRecord struct {
	Leader              string      `json:"leader"`
	Fields              []MARCField `json:"fields"`
	RecordID            string      `json:"record_id"`
	Title               string      `json:"title"`
	Author              string      `json:"author"`
	ISBN                string      `json:"isbn"`
	ISSN                string      `json:"issn"`
	Publisher           string      `json:"publisher"`
	Subject             string      `json:"subject"`
	Summary             string      `json:"summary"`
	TOC                 string      `json:"toc"`
	Edition             string      `json:"edition"`
	PhysicalDescription string      `json:"physical_description"`
	Series              string      `json:"series"`
	Notes               string      `json:"notes"`
	Holdings            []Holding   `json:"holdings"`
}

type MARCProfile struct {
	ISBNTag      string
	ISSNTag      string
	TitleTag     string
	AuthorTag    string
	PublisherTag string
	SubjectTag   string
}

var (
	ProfileMARC21 = MARCProfile{ISBNTag: "020", ISSNTag: "022", TitleTag: "245", AuthorTag: "100", PublisherTag: "260", SubjectTag: "650"}
	ProfileCNMARC = MARCProfile{ISBNTag: "010", ISSNTag: "011", TitleTag: "200", AuthorTag: "200", PublisherTag: "210", SubjectTag: "606"}
	ProfileUNIMARC = MARCProfile{ISBNTag: "010", ISSNTag: "011", TitleTag: "200", AuthorTag: "700", PublisherTag: "210", SubjectTag: "606"}
)

func ParseMARC(data []byte) (*MARCRecord, error) {
	if len(data) < 24 { return nil, fmt.Errorf("data too short") }
	if len(data) > 0 && data[0] == '{' {
		return ParseMARCJSON(string(data))
	}
	leader := string(data[:24])
	baseAddrStr := leader[12:17]
	baseAddr, err := strconv.Atoi(baseAddrStr)
	if err != nil { return nil, fmt.Errorf("bad base addr") }
	dirEnd := baseAddr - 1
	if dirEnd > len(data) || dirEnd < 24 { return nil, fmt.Errorf("bad directory") }
	directory := data[24:dirEnd]
	rec := &MARCRecord{Leader: leader}
	for i := 0; i < len(directory); i += 12 {
		if i+12 > len(directory) { break }
		entry := directory[i : i+12]
		tag := string(entry[:3])
		length, _ := strconv.Atoi(string(entry[3:7]))
		start, _ := strconv.Atoi(string(entry[7:12]))
		fieldStart, fieldEnd := baseAddr+start, baseAddr+start+length
		if fieldEnd > len(data) { continue }
		valData := data[fieldStart:fieldEnd]
		valData = bytes.TrimSuffix(valData, []byte{0x1e})
		rec.Fields = append(rec.Fields, MARCField{Tag: tag, Value: cleanSubfields(valData)})
	}
	rec.PopulateFriendlyFields()
	return rec, nil
}

func ParseMARCJSON(jsonStr string) (*MARCRecord, error) {
	var mj struct {
		Leader string                   `json:"leader"`
		Fields []map[string]interface{} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &mj); err != nil { return nil, err }

	rec := &MARCRecord{Leader: mj.Leader}
	for _, fMap := range mj.Fields {
		for tag, content := range fMap {
			val := ""
			switch v := content.(type) {
			case string:
				val = v
			case map[string]interface{}:
				if subs, ok := v["subfields"].([]interface{}); ok {
					var sb strings.Builder
					for _, s := range subs {
						if sm, ok := s.(map[string]interface{}); ok {
							for _, sv := range sm {
								if svs, ok := sv.(string); ok {
									sb.WriteString(" "); sb.WriteString(svs)
								}
							}
						}
					}
					val = strings.TrimSpace(sb.String())
				}
			}
			if val != "" {
				rec.Fields = append(rec.Fields, MARCField{Tag: tag, Value: val})
			}
		}
	}
	rec.PopulateFriendlyFields()
	return rec, nil
}

func (r *MARCRecord) PopulateFriendlyFields() {
	// Auto-detect profile based on fields? Default to MARC21 for now
	p := &ProfileMARC21
	r.RecordID = r.GetFieldByTag("001")
	r.Title = r.GetTitle(p)
	r.Author = r.GetAuthor(p)
	r.ISBN = r.GetISBN(p)
	r.ISSN = r.GetISSN(p)
	r.Publisher = r.GetPublisher(p)
	r.Subject = r.GetSubject(p)
	
	// Extended fields
	r.Summary = r.GetFieldByTag("520")
	r.TOC = r.GetFieldByTag("505")
	r.Edition = r.GetFieldByTag("250")
	r.PhysicalDescription = r.GetFieldByTag("300")
	
	// Series: Try 490, fallback to 830
	r.Series = r.GetFieldByTag("490")
	if r.Series == "" {
		r.Series = r.GetFieldByTag("830")
	}
	
	r.Notes = r.GetFieldByTag("500")
}

func cleanSubfields(data []byte) string {
	decoded := DecodeText(data)
	res := bytes.Buffer{}
	skip := false
	for _, r := range decoded {
		if r == 0x1f { skip = true; continue }
		if skip { skip = false; res.WriteRune(' '); continue }
		res.WriteRune(r)
	}
	return res.String()
}

func BuildMARC(profile *MARCProfile, id, title, author, isbn, publisher, pubYear, issn, subject string) []byte {
	if profile == nil { profile = &ProfileMARC21 }
	var db, dir bytes.Buffer
	addC := func(t, v string) {
		if v == "" { return }
		s := db.Len()
		db.WriteString(v); db.WriteByte(0x1e)
		dir.WriteString(fmt.Sprintf("%s%04d%05d", t, db.Len()-s, s))
	}
	addD := func(t string, subs map[string]string) {
		s := db.Len()
		for c, v := range subs {
			if v == "" { continue }
			db.WriteByte(0x1f); db.WriteByte(c[0]); db.WriteString(v)
		}
		db.WriteByte(0x1e)
		dir.WriteString(fmt.Sprintf("%s%04d%05d", t, db.Len()-s, s))
	}
	addC("001", id)
	addC("008", "260101s2026    xx      000 0 und d")
	addD(profile.ISBNTag, map[string]string{"a": isbn})
	addD(profile.ISSNTag, map[string]string{"a": issn})
	if profile.TitleTag == "200" { addD("200", map[string]string{"a": title, "f": author}) } else {
		addD("245", map[string]string{"a": title})
		if author != "" { addD("100", map[string]string{"a": author}) }
	}
	p := make(map[string]string)
	if profile.PublisherTag == "210" { p["c"], p["d"] = publisher, pubYear } else { p["b"], p["c"] = publisher, pubYear }
	if len(p) > 0 { addD(profile.PublisherTag, p) }
	
	if subject != "" {
		addD(profile.SubjectTag, map[string]string{"a": subject})
	}
	
	ba := 24 + dir.Len() + 1
	l := fmt.Sprintf("%05dnam a22%05d z 4500", ba+db.Len()+1, ba)
	return append(append(append([]byte(l), dir.Bytes()...), 0x1e), append(db.Bytes(), 0x1d)...)
}

func (r *MARCRecord) GetFieldByTag(tag string) string {
	for _, f := range r.Fields { if f.Tag == tag { return f.Value } }
	return ""
}
func (r *MARCRecord) GetTitle(p *MARCProfile) string {
	if p == nil { p = &ProfileMARC21 }
	return r.GetFieldByTag(p.TitleTag)
}
func (r *MARCRecord) GetAuthor(p *MARCProfile) string {
	if p == nil { p = &ProfileMARC21 }
	if p.TitleTag == "200" {
		val := r.GetFieldByTag("700")
		if val != "" { return val }
		val = r.GetFieldByTag("701")
		if val != "" { return val }
	}
	return r.GetFieldByTag(p.AuthorTag)
}
func (r *MARCRecord) GetISBN(p *MARCProfile) string {
	if p == nil { p = &ProfileMARC21 }
	raw := r.GetFieldByTag(p.ISBNTag)
	// Clean the ISBN using same logic as provider/utils.go
	// 1. Remove prefixes like "ISBN"
	// 2. Remove non-digit/non-X
	// Since we don't have the full prefix regex here easily, just do basic cleaning of non-alphanum
	// Actually, just removing non-digit/X is usually enough for the extracted subfield.
	
	// Split by space first, as often it's "ISBN ... (pbk.)"
	parts := strings.Fields(raw)
	if len(parts) > 0 {
		raw = parts[0]
	}
	
	return isbnCleanRegex.ReplaceAllString(raw, "")
}
func (r *MARCRecord) GetISSN(p *MARCProfile) string {
	if p == nil { p = &ProfileMARC21 }
	return r.GetFieldByTag(p.ISSNTag)
}
func (r *MARCRecord) GetPublisher(p *MARCProfile) string {
	if p == nil { p = &ProfileMARC21 }
	if p.TitleTag == "245" { 
		v := r.GetFieldByTag("264")
		if v != "" { return v }
		return r.GetFieldByTag("260")
	}
	return r.GetFieldByTag(p.PublisherTag)
}
func (r *MARCRecord) GetSubject(p *MARCProfile) string {
	if p == nil { p = &ProfileMARC21 }
	return r.GetFieldByTag(p.SubjectTag)
}
