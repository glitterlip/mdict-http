package dict

type Dict struct {
	ID               string                                  `json:"id"`
	Name             string                                  `json:"name"`
	Path             string                                  `json:"-"`
	Files            []string                                `json:"files"`
	Meta             map[string]interface{}                  `json:"meta"`
	Num              int64                                   `json:"num"`
	Dictionary       *Mdict                                  `json:"-"`
	AfterRecordFound func(dict *Dict, content string) string `json:"-"`
	Mdds             []*Mdict                                `json:"-"`
}

func (d *Dict) AddFile(id string) *Dict {
	d.Files = append(d.Files, id)
	return d
}
func (d *Dict) LookUp(word string) (string, error) {
	raw, err := d.Dictionary.Lookup(word)
	if err != nil {
		return "", err
	}

	result := string(raw)

	return result, nil
}
