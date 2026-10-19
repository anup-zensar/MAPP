package history

// CountsText says what came with a copied record, in the words a delete uses for what went with it: "with 2
// initiatives, 7 activities and 15 line items" (the details of a "copied" log entry: 004 plan *Activity
// log*, D-097, D-110). counts holds how many records of each kind were copied, by the kind's name; the
// kinds are named in the order they were registered, with the singular for 1, and kinds without records or
// unknown to history are left out. It is "" when nothing came with the record.
func (s *Service) CountsText(counts map[string]int) string {
	contents := []ContentCount{}
	for _, k := range s.kinds {
		n := counts[k.Name]
		if n <= 0 {
			continue
		}
		label := k.Plural
		if n == 1 {
			label = k.Singular
		}
		contents = append(contents, ContentCount{RecordType: k.Name, Label: label, Count: n})
	}
	return contentsText(contents)
}
